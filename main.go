package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"payment-middleware/internal/ably"
	"payment-middleware/internal/config"
	"payment-middleware/internal/handlers"
	"payment-middleware/internal/mapper"
	"payment-middleware/internal/store"
)

func main() {
	// Load configuration from environment/config file
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	log.Printf("Configuration loaded successfully")
	log.Printf("Server Port: %d", cfg.ServerPort)
	log.Printf("Timeout Duration: %v", cfg.TimeoutDuration)
	log.Printf("MID/TID Mappings: %d entries", len(cfg.MIDTIDMappings))

	// Initialize Redis client with retry logic
	redisConfig := store.RedisConfig{
		Host:         cfg.RedisHost,
		Port:         cfg.RedisPort,
		Password:     cfg.RedisPassword,
		DB:           cfg.RedisDB,
		MinIdleConns: cfg.RedisMinIdleConns,
		MaxConns:     cfg.RedisMaxConns,
	}
	redisClient, err := store.NewRedisClient(redisConfig)
	if err != nil {
		log.Fatalf("Failed to initialize Redis client: %v", err)
	}
	defer redisClient.Close()
	log.Printf("Redis client initialized successfully")

	// Initialize RedisMIDTIDMapper with Redis client
	midtidMapper := mapper.NewRedisMIDTIDMapper(redisClient.Client())
	log.Printf("RedisMIDTIDMapper initialized")

	// Initialize RedisTransactionStore with Redis client
	transactionStore := store.NewRedisTransactionStore(redisClient.Client())
	log.Printf("RedisTransactionStore initialized")

	// Initialize Ably client with API key and encryption secret
	ablyClient, err := ably.NewAblyClient(cfg.AblyAPIKey, cfg.EncryptionSecret)
	if err != nil {
		log.Fatalf("Failed to initialize Ably client: %v", err)
	}
	defer ablyClient.Close()
	log.Printf("Ably client initialized")

	// Set up Ably subscription for EDC responses with handler
	edcHandler := handlers.NewEDCResponseHandler(transactionStore)
	err = ablyClient.SubscribeToResponses(edcHandler.HandleEDCResponse)
	if err != nil {
		log.Fatalf("Failed to subscribe to EDC responses: %v", err)
	}
	log.Printf("Subscribed to EDC response channels")

	// Initialize HTTP router with all endpoints
	router := mux.NewRouter()

	// Add CORS middleware (must be first to handle preflight requests)
	router.Use(handlers.CORSMiddleware)

	// Add panic recovery middleware
	router.Use(handlers.PanicRecoveryMiddleware)

	// Initialize transaction handler
	transactionHandler := handlers.NewTransactionHandler(transactionStore, midtidMapper, ablyClient)

	// Initialize admin handler
	adminHandler := handlers.NewAdminHandler(midtidMapper, redisClient.Client())

	// Initialize health check handler
	healthHandler := handlers.NewHealthHandler(redisClient)

	// Register API endpoints
	router.HandleFunc("/api/v1/transaction", transactionHandler.HandleTransaction).Methods("POST", "OPTIONS")
	router.HandleFunc("/api/v1/transaction/status/{trx_id}", transactionHandler.HandleTransactionStatus).Methods("GET", "OPTIONS")

	// Register admin endpoints
	router.HandleFunc("/api/v1/admin/mapping", adminHandler.CreateOrUpdateMapping).Methods("POST", "OPTIONS")
	router.HandleFunc("/api/v1/admin/mapping", adminHandler.DeleteMapping).Methods("DELETE", "OPTIONS")
	router.HandleFunc("/api/v1/admin/migrate", func(w http.ResponseWriter, r *http.Request) {
		adminHandler.MigrateMapping(w, r, cfg.MIDTIDMappings)
	}).Methods("POST", "OPTIONS")
	router.HandleFunc("/api/v1/admin/transaction/{trx_id}/ttl", adminHandler.GetTransactionTTL).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/v1/admin/transaction/{trx_id}/extend-ttl", adminHandler.ExtendTransactionTTL).Methods("POST", "OPTIONS")

	// Health check endpoint
	router.HandleFunc("/health", healthHandler.CheckHealth).Methods("GET", "OPTIONS")

	log.Printf("HTTP routes registered")

	// Configure HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.ServerPort),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: cfg.TimeoutDuration + 15*time.Second, // Transaction timeout + buffer
		IdleTimeout:  120 * time.Second,
	}

	// Start HTTP server on configured port
	go func() {
		log.Printf("Starting Payment Middleware server on port %d", cfg.ServerPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Add graceful shutdown handling
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Attempt graceful shutdown
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}
