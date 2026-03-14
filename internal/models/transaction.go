package models

import "time"

// TransactionStatus represents the current state of a payment transaction
type TransactionStatus string

const (
	StatusPending TransactionStatus = "PENDING"
	StatusSuccess TransactionStatus = "SUCCESS"
	StatusFailed  TransactionStatus = "FAILED"
	StatusTimeout TransactionStatus = "TIMEOUT"
)

// Transaction represents a payment transaction with its state and lifecycle data
type Transaction struct {
	TrxID        string            `json:"trx_id"`
	Status       TransactionStatus `json:"status"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
	RequestData  PaymentRequest    `json:"request_data"`
	ResponseData *EDCResponse      `json:"response_data,omitempty"`
	NotifyChan   chan struct{}     `json:"-"` // Used for wait/notify mechanism
}
