# HTTP Client Library Specification

## Overview

A Go library providing a convenient HTTP client with interceptor/middleware support for making HTTP requests with JSON serialization.

## Architecture

### Core Components

```
+------------------+     +-------------------+     +------------------+
|     Client       |---->|  interTransport   |---->|  http.Transport  |
+------------------+     +-------------------+     +------------------+
        |                        |
        v                        v
  ClientOpt funcs           Interceptor chain
```

### Key Types

| Type | Description |
|------|-------------|
| `Client` | Main HTTP client wrapper with headers and base URL support |
| `ClientOpt` | Functional option pattern for configuring Client |
| `Handler` | Function type `func(*http.Request) (*http.Response, error)` |
| `Interceptor` | Middleware function `func(*http.Request, Handler) (*http.Response, error)` |
| `interTransport` | Custom http.RoundTripper that chains interceptors |
| `ErrorResponse` | Custom error type for non-2xx HTTP responses |

### Design Patterns

1. **Functional Options Pattern** - Used for Client configuration (`With*` functions)
2. **Interceptor/Middleware Pattern** - Chain of responsibility for request/response processing
3. **Transport Wrapper Pattern** - Wraps http.RoundTripper to inject interceptors

## Data Models

### Client Structure

```go
type Client struct {
    client  *http.Client       // underlying HTTP client (private)
    BaseURL *url.URL           // optional base URL for relative paths
    Headers map[string]string  // default headers for all requests
}
```

### Error Response

```go
type ErrorResponse struct {
    Response  *http.Response  // original HTTP response
    Message   string          // response body as string
    RequestID string          // optional request ID for tracing
}
```

## API Reference

### Package Functions

- `Get(url string, out interface{}) error` - Simple GET request
- `Post(url string, in, out interface{}) error` - Simple POST request

### Client Methods

- `NewClient(httpClient *http.Client) *Client` - Create client with defaults
- `New(httpClient *http.Client, opts ...ClientOpt) (*Client, error)` - Create configured client
- `(*Client) Get(url string, out interface{}) error` - GET with JSON decode
- `(*Client) Post(url string, in, out interface{}) error` - POST with JSON encode/decode
- `(*Client) Do(ctx context.Context, req *http.Request, v interface{}) (*http.Response, error)` - Execute request
- `(*Client) NewRequest(method, urlStr string, body interface{}) (*http.Request, error)` - Build request
- `(*Client) AddInterceptor(inter Interceptor) error` - Add middleware
- `(*Client) SetHeader(key, value string)` - Set default header
- `(*Client) SetAuthorization(auth string)` - Set auth header

### Configuration Options

- `WithBaseURL(bu string) ClientOpt` - Set base URL
- `WithUserAgent(ua string) ClientOpt` - Set User-Agent header
- `WithAuthorization(auth string) ClientOpt` - Set Authorization header
- `WithInterceptor(inter Interceptor) ClientOpt` - Add interceptor

### Built-in Interceptors

- `DumpInterceptor` - Logs request/response dumps for debugging
- `ResponseInterceptor` - Replaces NaN with null in JSON responses

## Request/Response Flow

```
1. Client.Do() called
   |
2. Request built with headers
   |
3. interTransport.RoundTrip()
   |
4. Interceptor chain executed (pre-request)
   |
5. Actual HTTP request via underlying transport
   |
6. Interceptor chain executed (post-response)
   |
7. Response status checked (CheckResponse)
   |
8. Response body decoded to target type
   |
9. Result returned
```

## Testing Strategy

### Test Types

1. **Unit Tests** - Test individual functions in isolation
2. **Integration Tests** - Test full request/response flow with httptest.Server

### Test Patterns

- Use `httptest.NewServer` for HTTP mocking
- Use `github.com/stretchr/testify/assert` for assertions
- Use `t.Run()` for subtests
- Use `t.Helper()` for test utility functions

### Running Tests

```bash
# All tests
go test ./... -v

# Single test
go test -run TestClient_Post -v

# With coverage
go test -cover ./...
```

## Dependencies

| Dependency | Purpose |
|------------|---------|
| `github.com/stretchr/testify` | Testing assertions (test only) |

## Version History

- v0.5.0 - Current version
- v0.0.1 - Initial release

## Future Considerations

- Add retry interceptor
- Add circuit breaker support
- Add request/response logging levels
- Add timeout configuration options
- Add connection pooling configuration
- Support for non-JSON content types
