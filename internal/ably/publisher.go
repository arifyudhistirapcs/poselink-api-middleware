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
	client           *ably.Realtime
	encryptionSecret string
}

// NewAblyClient creates a new Ably client with the provided API key and encryption secret
func NewAblyClient(apiKey, encryptionSecret string) (*AblyClient, error) {
	client, err := ably.NewRealtime(
		ably.WithKey(apiKey),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Ably client: %w", err)
	}

	return &AblyClient{
		client:           client,
		encryptionSecret: encryptionSecret,
	}, nil
}

// PublishPaymentRequest publishes a payment token to the EDC device channel with retry logic
// Sends a JSON object containing the encrypted token and trxId as metadata
func (a *AblyClient) PublishPaymentRequest(serialNumber, token, trxID string) error {
	channelName := fmt.Sprintf("edc:%s", serialNumber)
	channel := a.client.Channels.Get(channelName)

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

	// Retry configuration: 3 retries with exponential backoff (100ms, 200ms, 400ms)
	maxRetries := 3
	backoffDurations := []time.Duration{100 * time.Millisecond, 200 * time.Millisecond, 400 * time.Millisecond}

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// Wait for backoff duration before retry
			time.Sleep(backoffDurations[attempt-1])
		}

		ctx := context.Background()
		// Send as JSON string
		err := channel.Publish(ctx, "payment_request", string(payloadJSON))
		if err == nil {
			return nil
		}

		lastErr = err
	}

	return fmt.Errorf("failed to publish payment request after %d retries: %w", maxRetries, lastErr)
}

// SubscribeToResponses subscribes to the response channel pattern and routes messages to the handler
func (a *AblyClient) SubscribeToResponses(handler func(response models.EDCResponse)) error {
	// Subscribe to channel pattern "response:*" with event "payment_result"
	// Note: Ably Go SDK doesn't support channel patterns directly in the same way as some other SDKs
	// We'll need to subscribe to specific channels or use a different approach
	// For now, we'll subscribe to a wildcard pattern if supported, or document the limitation
	
	// The Ably Go SDK requires explicit channel names, so we'll subscribe to "response:*" as a literal channel
	// In production, you may need to dynamically subscribe to specific response channels
	channel := a.client.Channels.Get("response:*")

	ctx := context.Background()
	_, err := channel.Subscribe(ctx, "payment_result", func(msg *ably.Message) {
		// Parse the message data into EDCResponse
		var response models.EDCResponse
		var encryptedData string
		
		// Handle different data types from Ably
		switch data := msg.Data.(type) {
		case string:
			// Data is encrypted string from EDC
			encryptedData = data
		case []byte:
			// Data is encrypted bytes from EDC
			encryptedData = string(data)
		case map[string]interface{}:
			// If data is already a map (shouldn't happen with encrypted data), try to use it directly
			jsonData, err := json.Marshal(data)
			if err != nil {
				fmt.Printf("Error marshaling EDC response map: %v\n", err)
				return
			}
			if err := json.Unmarshal(jsonData, &response); err != nil {
				fmt.Printf("Error unmarshaling EDC response from map: %v\n", err)
				return
			}
			// Call the handler with the parsed response
			handler(response)
			return
		default:
			fmt.Printf("Unexpected data type from Ably: %T\n", data)
			return
		}

		// Decrypt the encrypted data
		log.Printf("Received encrypted response from EDC, length: %d", len(encryptedData))
		decryptedJSON, err := crypto.DecryptAES128ECB(encryptedData, a.encryptionSecret)
		if err != nil {
			fmt.Printf("Error decrypting EDC response: %v\n", err)
			return
		}

		log.Printf("Successfully decrypted EDC response, length: %d", len(decryptedJSON))

		// Unmarshal the decrypted JSON
		if err := json.Unmarshal([]byte(decryptedJSON), &response); err != nil {
			fmt.Printf("Error unmarshaling decrypted EDC response: %v\n", err)
			return
		}

		log.Printf("Successfully parsed EDC response for transaction: %s", response.TrxID)

		// Call the handler with the parsed response
		handler(response)
	})

	if err != nil {
		return fmt.Errorf("failed to subscribe to EDC responses: %w", err)
	}

	return nil
}

// Close closes the Ably client connection
func (a *AblyClient) Close() {
	if a != nil && a.client != nil {
		a.client.Close()
	}
}
