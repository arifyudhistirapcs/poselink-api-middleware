package ably

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/ably/ably-go/ably"
	"payment-middleware/internal/crypto"
	"payment-middleware/internal/models"
)

// AblyPublisher defines the interface for publishing payment requests and subscribing to EDC responses
type AblyPublisher interface {
	// PublishPaymentRequest publishes a payment token to the EDC device channel with transaction ID metadata
	PublishPaymentRequest(serialNumber, token, trxID string) error
	
	// SubscribeToResponses subscribes to EDC response channels and handles incoming responses
	SubscribeToResponses(handler func(response models.EDCResponse)) error
}

// AblyClient implements the AblyPublisher interface using the Ably SDK
type AblyClient struct {
	realtimeClient   *ably.Realtime
	restClient       *ably.REST
	encryptionSecret string
}

// NewAblyClient creates a new Ably client with the provided API key and encryption secret
func NewAblyClient(apiKey, encryptionSecret string) (*AblyClient, error) {
	// REST client for publishing (ensures JSON encoding compatible with Android SDK)
	restClient, err := ably.NewREST(
		ably.WithKey(apiKey),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Ably REST client: %w", err)
	}

	// Realtime client for subscribing to responses
	realtimeClient, err := ably.NewRealtime(
		ably.WithKey(apiKey),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Ably Realtime client: %w", err)
	}

	return &AblyClient{
		realtimeClient:   realtimeClient,
		restClient:       restClient,
		encryptionSecret: encryptionSecret,
	}, nil
}

// PublishPaymentRequest publishes a payment token to the EDC device channel via REST API.
// Uses REST client to ensure JSON encoding compatible with Android Ably SDK.
func (a *AblyClient) PublishPaymentRequest(serialNumber, token, trxID string) error {
	channelName := fmt.Sprintf("edc:%s", serialNumber)
	channel := a.restClient.Channels.Get(channelName)

	// Create message payload with token and trxId metadata
	payload := map[string]string{
		"token":  token,
		"trx_id": trxID,
	}

	// Marshal payload to JSON string
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Retry configuration: 3 retries with exponential backoff
	maxRetries := 3
	backoffDurations := []time.Duration{100 * time.Millisecond, 200 * time.Millisecond, 400 * time.Millisecond}

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(backoffDurations[attempt-1])
		}

		ctx := context.Background()
		err := channel.Publish(ctx, "payment_request", string(payloadJSON))
		if err == nil {
			log.Printf("Published payment request to channel %s (attempt %d)", channelName, attempt+1)
			return nil
		}

		lastErr = err
		log.Printf("Failed to publish to %s (attempt %d): %v", channelName, attempt+1, err)
	}

	return fmt.Errorf("failed to publish payment request after %d retries: %w", maxRetries, lastErr)
}

// SubscribeToResponses subscribes to the response channel and routes messages to the handler
func (a *AblyClient) SubscribeToResponses(handler func(response models.EDCResponse)) error {
	channel := a.realtimeClient.Channels.Get("response:*")

	ctx := context.Background()
	_, err := channel.Subscribe(ctx, "payment_result", func(msg *ably.Message) {
		var response models.EDCResponse
		var encryptedData string
		
		switch data := msg.Data.(type) {
		case string:
			encryptedData = data
		case []byte:
			encryptedData = string(data)
		case map[string]interface{}:
			jsonData, err := json.Marshal(data)
			if err != nil {
				fmt.Printf("Error marshaling EDC response map: %v\n", err)
				return
			}
			if err := json.Unmarshal(jsonData, &response); err != nil {
				fmt.Printf("Error unmarshaling EDC response from map: %v\n", err)
				return
			}
			handler(response)
			return
		default:
			fmt.Printf("Unexpected data type from Ably: %T\n", data)
			return
		}

		log.Printf("Received encrypted response from EDC, length: %d", len(encryptedData))
		decryptedJSON, err := crypto.DecryptAES128ECB(encryptedData, a.encryptionSecret)
		if err != nil {
			fmt.Printf("Error decrypting EDC response: %v\n", err)
			return
		}

		log.Printf("Successfully decrypted EDC response, length: %d", len(decryptedJSON))

		if err := json.Unmarshal([]byte(decryptedJSON), &response); err != nil {
			fmt.Printf("Error unmarshaling decrypted EDC response: %v\n", err)
			return
		}

		log.Printf("Successfully parsed EDC response for transaction: %s", response.TrxID)
		handler(response)
	})

	if err != nil {
		return fmt.Errorf("failed to subscribe to EDC responses: %w", err)
	}

	return nil
}

// Close closes both Ably clients
func (a *AblyClient) Close() {
	if a != nil && a.realtimeClient != nil {
		a.realtimeClient.Close()
	}
}
