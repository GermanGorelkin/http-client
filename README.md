# HTTP Client Library for Go

A convenient HTTP client library for Go with interceptor/middleware support, JSON serialization, multipart uploads, and functional configuration options.

## Features

- **Interceptor/Middleware Support**: Chain request/response handlers for logging, retries, authentication, etc.
- **Built-in Retry Interceptor**: Automatic retry for transient errors with exponential backoff and jitter
- **Functional Options Pattern**: Clean configuration with `With*` functions
- **JSON Serialization**: Automatic JSON encoding/decoding for request/response bodies
- **Multipart File Upload**: Support for multipart/form-data file uploads
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

### Multipart File Upload

```go
// Create a multipart form
form := http_client.NewMultipartForm()
form.AddField("description", "Profile picture")
form.AddField("user_id", "123")

// Add file from io.Reader (e.g., from os.Open, bytes.NewReader, etc.)
file, err := os.Open("profile.jpg")
if err != nil {
    log.Fatal(err)
}
defer file.Close()

form.AddFile("attachment", "profile.jpg", file)

// Upload with multipart POST
var resp struct {
    Success bool   `json:"success"`
    URL     string `json:"url"`
}

err = client.PostMultipart("https://api.example.com/upload", form, &resp)
if err != nil {
    fmt.Printf("Error: %v\n", err)
    return
}

fmt.Printf("Upload successful: %s\n", resp.URL)
```

### Multiple Files Upload

```go
form := http_client.NewMultipartForm()
form.AddField("album", "vacation")

// Add multiple files under same field name
form.AddFile("photos", "photo1.jpg", file1)
form.AddFile("photos", "photo2.jpg", file2)
form.AddFile("photos", "photo3.jpg", file3)

err := client.PostMultipart("https://api.example.com/upload", form, nil)
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

### MultipartForm Methods

| Method | Description | Example |
|--------|-------------|---------|
| `NewMultipartForm()` | Creates empty multipart form | `form := NewMultipartForm()` |
| `AddField(name, value)` | Adds text field (can be called multiple times for same name) | `form.AddField("description", "my file")` |
| `AddFile(fieldName, fileName, reader)` | Adds file from `io.Reader` | `form.AddFile("file", "photo.jpg", reader)` |

### Client Methods for Multipart

| Method | Description | Example |
|--------|-------------|---------|
| `NewMultipartRequest(method, url, form)` | Creates multipart HTTP request | `req, err := client.NewMultipartRequest("POST", "/upload", form)` |
| `PostMultipart(url, form, out)` | Sends multipart POST and decodes response | `err := client.PostMultipart("/upload", form, &resp)` |

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

// Create multipart form
form := http_client.NewMultipartForm()
form.AddField("key", "value")
// ... add files
```

### Creating Multipart Requests Manually

```go
// For more control, create request manually
form := http_client.NewMultipartForm()
form.AddField("description", "document")
form.AddFile("file", "document.pdf", fileReader)

req, err := client.NewMultipartRequest("POST", "/upload", form)
if err != nil {
    log.Fatal(err)
}

// Add custom headers to multipart request
req.Header.Set("X-Custom-Header", "value")

// Execute with context
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

resp, err := client.Do(ctx, req, &response)
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

### 3. Multipart Upload Memory Considerations

```go
// For small files (<10MB), use standard multipart upload:
form := http_client.NewMultipartForm()
form.AddField("description", "small file")
form.AddFile("file", "document.pdf", fileReader)
err := client.PostMultipart("/upload", form, &resp)

// For large files, consider:
// 1. Chunking the file into smaller parts
// 2. Using streaming uploads (future feature)
// 3. Compressing before upload
// 4. Using direct file upload APIs if available

// Current implementation buffers entire form in memory.
// Monitor memory usage when uploading multiple large files.

// Future streaming API example (planned):
// ```go
// // Streaming upload for large files without buffering
// err := client.PostMultipartStream("/upload", func(w *multipart.Writer) error {
//     // Add fields
//     if err := w.WriteField("description", "large file"); err != nil {
//         return err
//     }
//     
//     // Stream file directly
//     part, err := w.CreateFormFile("file", "large-video.mp4")
//     if err != nil {
//         return err
//     }
//     
//     // Stream from file without buffering entire content
//     file, err := os.Open("large-video.mp4")
//     if err != nil {
//         return err
//     }
//     defer file.Close()
//     
//     _, err = io.Copy(part, file)
//     return err
// }, &resp)
// ```
```

### 4. Validate Inputs

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

### v0.7.0
- **Added Multipart Support**: Robust support for multipart/form-data uploads
  - Added `PostMultipart` method for file uploads
  - Added `MultipartForm` builder for complex requests
  - Support for multiple files and fields
  - Support for uploading from `io.Reader`

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