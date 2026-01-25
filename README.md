# HTTP Client Library for Go

A convenient HTTP client library for Go with interceptor/middleware support, JSON serialization, and functional configuration options.

## Features

- **Interceptor/Middleware Support**: Chain request/response handlers for logging, retries, authentication, etc.
- **Built-in Retry Interceptor**: Automatic retry for transient errors with exponential backoff and jitter
- **Functional Options Pattern**: Clean configuration with `With*` functions
- **JSON Serialization**: Automatic JSON encoding/decoding for request/response bodies
- **Error Handling**: Custom error types for HTTP errors with response details
- **Base URL Support**: Set a base URL for all relative requests
- **Header Management**: Default headers for all requests
- **Test Utilities**: Built-in interceptors for debugging and response processing

## Installation

```bash
go get github.com/germangorelkin/http-client
```

## Quick Start

### Basic Usage

```go
package main

import (
    "fmt"
    "github.com/germangorelkin/http-client"
)

func main() {
    // Create a client with default configuration
    client := http_client.NewClient(nil)

    // GET request with JSON decoding
    user := struct {
        Name string `json:"name"`
        Age  int    `json:"age"`
    }{}
    
    err := client.Get("https://api.example.com/users/1", &user)
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }
    
    fmt.Printf("User: %s, Age: %d\n", user.Name, user.Age)
}
```

### POST Request with JSON

```go
// POST request with JSON encoding
newUser := struct {
    Name string `json:"name"`
    Age  int    `json:"age"`
}{
    Name: "John",
    Age:  30,
}

createdUser := struct {
    ID   string `json:"id"`
    Name string `json:"name"`
    Age  int    `json:"age"`
}{}

err := client.Post("https://api.example.com/users", newUser, &createdUser)
if err != nil {
    fmt.Printf("Error: %v\n", err)
    return
}

fmt.Printf("Created user ID: %s\n", createdUser.ID)
```

## Configuration Options

### Creating a Configured Client

```go
client, err := http_client.New(nil,
    http_client.WithBaseURL("https://api.example.com/v1"),
    http_client.WithUserAgent("my-app/1.0"),
    http_client.WithAuthorization("Bearer token123"),
    http_client.WithInterceptor(http_client.DefaultRetryInterceptor()),
    http_client.WithInterceptor(http_client.DumpInterceptor),
)
if err != nil {
    log.Fatal(err)
}
```

### Available Options

| Option | Description | Example |
|--------|-------------|---------|
| `WithBaseURL` | Sets base URL for relative paths | `WithBaseURL("https://api.example.com")` |
| `WithUserAgent` | Sets User-Agent header | `WithUserAgent("my-app/1.0")` |
| `WithAuthorization` | Sets Authorization header | `WithAuthorization("Bearer token")` |
| `WithInterceptor` | Adds an interceptor to the chain | `WithInterceptor(myInterceptor)` |

### RetryInterceptor Configuration Options

| Option | Description | Default |
|--------|-------------|---------|
| `WithMaxRetries` | Maximum retry attempts | 3 |
| `WithBackoff` | Base and maximum delay for exponential backoff | 100ms base, 1s max |
| `WithRetryOnStatus` | HTTP status codes that trigger retry | [408, 429, 500, 502, 503, 504] |
| `WithRetryMethods` | HTTP methods safe to retry | ["GET", "HEAD", "PUT", "DELETE", "OPTIONS"] |
| `WithRetryOnError` | Custom function to determine retryable errors | Network errors and context deadline |

### Runtime Configuration

```go
client := http_client.NewClient(nil)

// Set headers
client.SetHeader("X-Custom-Header", "value")
client.SetAuthorization("Bearer new-token")

// Add interceptors
client.AddInterceptor(myInterceptor)
```

## Interceptors

Interceptors are middleware functions that can intercept and modify HTTP requests and responses. They follow the chain of responsibility pattern.

### Built-in Interceptors

#### DumpInterceptor
Logs complete request and response dumps for debugging:

```go
client, err := http_client.New(nil,
    http_client.WithInterceptor(http_client.DumpInterceptor),
)
```

#### ResponseInterceptor
Replaces `NaN` with `null` in JSON responses to fix invalid JSON:

```go
client, err := http_client.New(nil,
    http_client.WithInterceptor(http_client.ResponseInterceptor),
)
```

#### RetryInterceptor
Automatically retries requests on transient errors with exponential backoff:

```go
// Default retry interceptor (3 retries, exponential backoff)
client, err := http_client.New(nil,
    http_client.WithInterceptor(http_client.DefaultRetryInterceptor()),
)

// Custom retry configuration
retryInterceptor := http_client.NewRetryInterceptor(
    http_client.WithMaxRetries(5),
    http_client.WithBackoff(100*time.Millisecond, 2*time.Second),
    http_client.WithRetryOnStatus([]int{408, 429, 500, 502, 503, 504}),
    http_client.WithRetryMethods([]string{"GET", "HEAD", "PUT", "DELETE", "OPTIONS"}),
)

client, err := http_client.New(nil,
    http_client.WithInterceptor(retryInterceptor),
)
```

**Default Retry Behavior:**
- **Max retries**: 3 attempts
- **Backoff**: Exponential with jitter (100ms, 200ms, 400ms)
- **Retry on status codes**: 408, 429, 500, 502, 503, 504
- **Retry on errors**: Network errors and context deadline exceeded
- **Safe methods**: GET, HEAD, PUT, DELETE, OPTIONS (POST, PATCH are not retried)

### Creating Custom Interceptors

```go
func LoggingInterceptor(req *http.Request, handler http_client.Handler) (*http.Response, error) {
    // Log request
    log.Printf("Request: %s %s", req.Method, req.URL)
    
    // Call next handler
    resp, err := handler(req)
    
    // Log response
    if err == nil {
        log.Printf("Response: %d %s", resp.StatusCode, resp.Status)
    }
    
    return resp, err
}

func AuthInterceptor(token string) http_client.Interceptor {
    return func(req *http.Request, handler http_client.Handler) (*http.Response, error) {
        req.Header.Set("Authorization", "Bearer "+token)
        return handler(req)
    }
}

// Usage
client, err := http_client.New(nil,
    http_client.WithInterceptor(LoggingInterceptor),
    http_client.WithInterceptor(AuthInterceptor("my-token")),
)
```

### Interceptor Chain Example

```go
// Example of a custom metrics interceptor
func MetricsInterceptor(metricsClient *MetricsClient) http_client.Interceptor {
    return func(req *http.Request, handler http_client.Handler) (*http.Response, error) {
        start := time.Now()
        
        resp, err := handler(req)
        
        duration := time.Since(start)
        statusCode := 0
        if resp != nil {
            statusCode = resp.StatusCode
        }
        
        metricsClient.RecordRequest(req.Method, req.URL.Path, statusCode, duration, err)
        
        return resp, err
    }
}

// Usage with built-in and custom interceptors
client, err := http_client.New(nil,
    http_client.WithInterceptor(http_client.DefaultRetryInterceptor()),
    http_client.WithInterceptor(MetricsInterceptor(metrics)),
    http_client.WithInterceptor(http_client.DumpInterceptor),
)
```

## Error Handling

### HTTP Error Responses

Non-2xx status codes return an `ErrorResponse`:

```go
err := client.Get("https://api.example.com/not-found", nil)
if err != nil {
    if errRes, ok := err.(*http_client.ErrorResponse); ok {
        fmt.Printf("HTTP Error: %d\n", errRes.Response.StatusCode)
        fmt.Printf("Message: %s\n", errRes.Message)
        fmt.Printf("Request: %s %s\n", 
            errRes.Response.Request.Method, 
            errRes.Response.Request.URL)
    }
}
```

### ErrorResponse Structure

```go
type ErrorResponse struct {
    Response  *http.Response  // Original HTTP response
    Message   string          // Response body as string
    RequestID string          // Optional request ID for tracing
}
```

## Advanced Usage

### Custom HTTP Client

```go
customClient := &http.Client{
    Timeout: 30 * time.Second,
    Transport: &http.Transport{
        MaxIdleConns:        100,
        IdleConnTimeout:     90 * time.Second,
        TLSHandshakeTimeout: 10 * time.Second,
    },
}

client := http_client.NewClient(customClient)
```

### Context Support

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

req, err := client.NewRequest("GET", "/users/1", nil)
if err != nil {
    log.Fatal(err)
}

resp, err := client.Do(ctx, req, &user)
if err != nil {
    if ctx.Err() == context.DeadlineExceeded {
        log.Println("Request timed out")
    }
}
```

### Raw Response Handling

```go
req, err := client.NewRequest("GET", "/users/1", nil)
if err != nil {
    log.Fatal(err)
}

resp, err := client.Do(context.Background(), req, nil)
if err != nil {
    log.Fatal(err)
}
defer resp.Body.Close()

// Access raw response
fmt.Printf("Status: %s\n", resp.Status)
fmt.Printf("Headers: %v\n", resp.Header)
```

## Testing

### Using httptest.Server

```go
func TestClientIntegration(t *testing.T) {
    ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        fmt.Fprintln(w, `{"id": "123", "name": "Test User"}`)
    }))
    defer ts.Close()

    client := http_client.NewClient(nil)
    
    user := struct {
        ID   string `json:"id"`
        Name string `json:"name"`
    }{}
    
    err := client.Get(ts.URL, &user)
    assert.NoError(t, err)
    assert.Equal(t, "123", user.ID)
    assert.Equal(t, "Test User", user.Name)
}
```

### Mocking Interceptors

```go
func TestInterceptor(t *testing.T) {
    var requestLogged bool
    var responseLogged bool
    
    loggingInterceptor := func(req *http.Request, handler http_client.Handler) (*http.Response, error) {
        requestLogged = true
        resp, err := handler(req)
        if err == nil {
            responseLogged = true
        }
        return resp, err
    }
    
    client, err := http_client.New(nil,
        http_client.WithInterceptor(loggingInterceptor),
    )
    assert.NoError(t, err)
    
    // Test with httptest server...
    assert.True(t, requestLogged)
    assert.True(t, responseLogged)
}
```

## Best Practices

### 1. Reuse Client Instances

```go
// Good: Create once, reuse
var apiClient *http_client.Client

func init() {
    var err error
    apiClient, err = http_client.New(nil,
        http_client.WithBaseURL("https://api.example.com"),
        http_client.WithUserAgent("my-service"),
    )
    if err != nil {
        log.Fatal(err)
    }
}
```

### 2. Handle Response Body Closure

```go
req, err := client.NewRequest("GET", "/data", nil)
if err != nil {
    return err
}

resp, err := client.Do(ctx, req, &data)
if err != nil {
    return err
}
// Response body is automatically closed by client.Do()
```

### 3. Validate Inputs

```go
func GetUser(id string) (*User, error) {
    if id == "" {
        return nil, errors.New("user ID cannot be empty")
    }
    
    user := &User{}
    err := client.Get(fmt.Sprintf("/users/%s", id), user)
    if err != nil {
        return nil, fmt.Errorf("failed to get user: %w", err)
    }
    
    return user, nil
}
```

## API Reference

### Package Functions

- `Get(url string, out any) error` - Simple GET request
- `Post(url string, in, out any) error` - Simple POST request
- `DoRequestWithClient(ctx context.Context, client *http.Client, req *http.Request) (*http.Response, error)` - Low-level request execution
- `CheckResponse(r *http.Response) error` - Check HTTP response status
- `DefaultRetryInterceptor() Interceptor` - Retry interceptor with default configuration
- `NewRetryInterceptor(opts ...RetryOpt) Interceptor` - Configurable retry interceptor

### Client Methods

- `NewClient(httpClient *http.Client) *Client` - Create client with defaults
- `New(httpClient *http.Client, opts ...ClientOpt) (*Client, error)` - Create configured client
- `(*Client) Get(url string, out any) error` - GET with JSON decode
- `(*Client) Post(url string, in, out any) error` - POST with JSON encode/decode
- `(*Client) Do(ctx context.Context, req *http.Request, v any) (*http.Response, error)` - Execute request
- `(*Client) NewRequest(method, urlStr string, body any) (*http.Request, error)` - Build request
- `(*Client) AddInterceptor(inter Interceptor) error` - Add middleware
- `(*Client) SetHeader(key, value string)` - Set default header
- `(*Client) SetAuthorization(auth string)` - Set auth header

### Types

- `type Client struct` - Main HTTP client
- `type ClientOpt func(*Client) error` - Functional option
- `type Handler func(*http.Request) (*http.Response, error)` - Request handler
- `type Interceptor func(*http.Request, Handler) (*http.Response, error)` - Middleware
- `type ErrorResponse struct` - HTTP error response
- `type RetryConfig struct` - Configuration for retry behavior
- `type RetryOpt func(*RetryConfig)` - Functional option for retry configuration

### Retry Configuration Functions

- `WithMaxRetries(n int) RetryOpt` - Set maximum retry attempts
- `WithBackoff(base, max time.Duration) RetryOpt` - Set exponential backoff delays
- `WithRetryOnStatus(codes []int) RetryOpt` - Set HTTP status codes that trigger retry
- `WithRetryMethods(methods []string) RetryOpt` - Set HTTP methods safe to retry
- `WithRetryOnError(fn func(error) bool) RetryOpt` - Set custom error retry logic

## Contributing

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Ensure all tests pass: `go test ./...`
5. Submit a pull request

### Running Tests

```bash
# Run all tests
go test ./... -v

# Run specific test
go test -run TestClient_Get -v

# Run with coverage
go test -cover ./...

# Run with race detector
go test -race ./...
```

### Code Style

- Follow standard Go conventions
- Use `gofmt` for formatting
- Add tests for new functionality
- Update documentation for API changes

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Changelog

### v0.6.0
- **Added RetryInterceptor**: Automatic retry for transient errors with exponential backoff
  - `DefaultRetryInterceptor()` with sensible defaults
  - `NewRetryInterceptor(opts ...RetryOpt)` for custom configuration
  - Configurable retry count, backoff, status codes, and methods
  - Jitter added to prevent thundering herd
  - Safe request cloning for retry attempts

### v0.5.0
- Added interceptor/middleware support
- Added functional options pattern
- Improved error handling with ErrorResponse
- Added base URL support
- Added header management

### v0.0.1
- Initial release with basic HTTP client functionality