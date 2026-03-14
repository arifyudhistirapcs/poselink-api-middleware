# Payment Middleware

A high-performance Go-based bridge service that enables communication between Point-of-Sale (POS) systems and Android EDC (Electronic Data Capture) devices using Ably Realtime messaging.

## Features

- Asynchronous payment transaction processing via Ably
- Synchronous response with 60-second timeout
- Transaction status polling endpoint
- Thread-safe concurrent transaction handling
- Graceful shutdown support
- Panic recovery middleware

## Prerequisites

- Go 1.22 or higher
- Ably API key

## Configuration

The service is configured via environment variables:

- `ABLY_API_KEY` (required): Your Ably API key
- `SERVER_PORT` (optional, default: 8080): HTTP server port
- `TIMEOUT_DURATION` (optional, default: 60): Transaction timeout in seconds
- `MIDTID_MAPPINGS` (optional): JSON string mapping MID/TID to serial numbers

Example:
```bash
export ABLY_API_KEY="your_ably_api_key"
export SERVER_PORT="8080"
export TIMEOUT_DURATION="60"
export MIDTID_MAPPINGS='{"M001:T001":"SN12345","M002:T002":"SN67890"}'
```

## Building

```bash
go build -o payment-middleware
```

## Running

```bash
./payment-middleware
```

## API Endpoints

### POST /api/v1/transaction

Initiate a payment transaction.

**Request:**
```json
{
  "token": "payment_token_string",
  "mid": "merchant_id",
  "tid": "terminal_id",
  "trx_id": "unique_transaction_id"
}
```

**Response (200 OK):**
```json
{
  "acq_mid": "string",
  "acq_tid": "string",
  "status": "success",
  "trx_id": "unique_transaction_id",
  ...
}
```

**Error Responses:**
- 400: Invalid request (missing trx_id)
- 404: Unknown MID/TID combination
- 408: Transaction timeout
- 503: Ably connection error

### GET /api/v1/transaction/status/{trx_id}

Poll transaction status.

**Response (200 OK):**
```json
{
  "status": "PENDING|SUCCESS|FAILED|TIMEOUT",
  "data": { /* EDC response if available */ }
}
```

**Error Response:**
- 404: Transaction not found

### GET /health

Health check endpoint.

**Response (200 OK):**
```json
{
  "status": "ok"
}
```

## Testing

Run all tests:
```bash
go test ./...
```

Run tests with race detector:
```bash
go test -race ./...
```

## Architecture

The service consists of the following components:

- **HTTP Server**: Handles REST API requests
- **Transaction Store**: In-memory storage using sync.Map
- **MID/TID Mapper**: Maps merchant/terminal IDs to device serial numbers
- **Ably Client**: Publishes requests and subscribes to EDC responses
- **Handlers**: Process transactions and EDC responses

## Graceful Shutdown

The service handles SIGINT and SIGTERM signals for graceful shutdown with a 30-second timeout.
