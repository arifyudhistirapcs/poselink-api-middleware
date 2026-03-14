package handlers

// Example usage documentation for TransactionHandler and EDCResponseHandler
//
// To use the handlers in your HTTP server:
//
// 1. Initialize dependencies:
//    store := store.NewSyncMapStore()
//    mapper := mapper.NewInMemoryMapper(map[string]string{
//        "M123:T456": "SN789",
//    })
//    ablyClient, _ := ably.NewAblyClient("your-api-key")
//
// 2. Create handlers:
//    transactionHandler := handlers.NewTransactionHandler(store, mapper, ablyClient)
//    edcResponseHandler := handlers.NewEDCResponseHandler(store)
//
// 3. Subscribe to EDC responses:
//    err := ablyClient.SubscribeToResponses(edcResponseHandler.HandleEDCResponse)
//    if err != nil {
//        log.Fatalf("Failed to subscribe to EDC responses: %v", err)
//    }
//
// 4. Register HTTP endpoints:
//    router := mux.NewRouter()
//    router.HandleFunc("/api/v1/transaction", transactionHandler.HandleTransaction).Methods("POST")
//    router.HandleFunc("/api/v1/transaction/status/{trx_id}", transactionHandler.HandleTransactionStatus).Methods("GET")
//
// 5. Start server:
//    http.ListenAndServe(":8080", router)
//
// The transaction handler will:
// - Parse and validate the incoming payment request
// - Map MID/TID to serial number
// - Store transaction with PENDING status
// - Publish payment token to Ably
// - Wait up to 60 seconds for EDC response
// - Return 200 with EDC response or 408 on timeout
//
// The EDC response handler will:
// - Parse incoming Ably messages into EDCResponse struct
// - Look up transaction by trx_id
// - Update transaction state to SUCCESS or FAILED based on status
// - Store complete EDC response data
// - Signal notification channel to unblock waiting request handler
//
// The transaction status handler will:
// - Extract trx_id from URL path parameter
// - Look up transaction in TransactionStore
// - Return 404 if transaction not found
// - Return 200 with StatusResponse containing status and data
// - Include EDC response data only if status is SUCCESS or FAILED
// - Ensure status is one of: PENDING, SUCCESS, FAILED, TIMEOUT
