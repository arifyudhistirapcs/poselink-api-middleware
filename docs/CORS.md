# CORS Configuration

The middleware includes CORS (Cross-Origin Resource Sharing) support to allow web applications from different origins to access the API.

## CORS Headers

The following CORS headers are automatically added to all responses:

- `Access-Control-Allow-Origin: *` - Allows requests from any origin
- `Access-Control-Allow-Methods: GET, POST, PUT, DELETE, OPTIONS` - Allowed HTTP methods
- `Access-Control-Allow-Headers: Content-Type, Authorization, X-Requested-With` - Allowed request headers
- `Access-Control-Max-Age: 3600` - Preflight cache duration (1 hour)

## Preflight Requests

The middleware automatically handles OPTIONS preflight requests for all endpoints. Browsers send these requests before making actual cross-origin requests.

## Testing CORS

### Test Preflight Request
```bash
curl -i -X OPTIONS http://localhost:8080/api/v1/transaction \
  -H "Origin: http://example.com" \
  -H "Access-Control-Request-Method: POST" \
  -H "Access-Control-Request-Headers: Content-Type"
```

### Test Actual Request with CORS
```bash
curl -i -X POST http://localhost:8080/api/v1/transaction \
  -H "Origin: http://example.com" \
  -H "Content-Type: application/json" \
  -d '{"token":"...","mid":"...","tid":"...","trx_id":"..."}'
```

## Security Considerations

**Current Configuration**: `Access-Control-Allow-Origin: *` allows requests from any origin.

**For Production**: Consider restricting to specific origins:

```go
// In internal/handlers/cors.go
w.Header().Set("Access-Control-Allow-Origin", "https://your-domain.com")
```

Or use environment variable for configuration:

```go
allowedOrigin := os.Getenv("CORS_ALLOWED_ORIGIN")
if allowedOrigin == "" {
    allowedOrigin = "*"
}
w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
```

## Implementation

CORS is implemented as middleware in `internal/handlers/cors.go` and applied globally to all routes in `main.go`.
