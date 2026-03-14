package models

// PaymentRequest represents the incoming payment request from POS system
type PaymentRequest struct {
	Token string `json:"token"`
	MID   string `json:"mid"`
	TID   string `json:"tid"`
	TrxID string `json:"trx_id"`
}
