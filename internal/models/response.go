package models

// EDCResponse represents the response from the EDC device
type EDCResponse struct {
	AcqMID          string `json:"acq_mid"`
	AcqTID          string `json:"acq_tid"`
	Action          string `json:"action"`
	Amount          string `json:"amount"`
	Approval        string `json:"approval"`
	BatchNumber     string `json:"batch_number"`
	CardCategory    string `json:"card_category"`
	CardName        string `json:"card_name"`
	CardType        string `json:"card_type"`
	EDCAddress      string `json:"edc_address"`
	IsCredit        string `json:"is_credit"`
	IsOffUs         string `json:"is_off_us"`
	Method          string `json:"method"`
	Msg             string `json:"msg"`
	PAN             string `json:"pan"`
	Periode         string `json:"periode"`
	Plan            string `json:"plan"`
	POSAddress      string `json:"pos_address"`
	RC              string `json:"rc"`
	ReferenceNumber string `json:"reference_number"`
	Status          string `json:"status"`
	TraceNumber     string `json:"trace_number"`
	TransactionDate string `json:"transaction_date"`
	TrxID           string `json:"trx_id"`
}

// StatusResponse represents the response for transaction status queries
type StatusResponse struct {
	Status string       `json:"status"`
	Data   *EDCResponse `json:"data,omitempty"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error"`
}
