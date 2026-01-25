# AGENTS.md - AI Coding Agent Guidelines

This document provides guidelines for AI coding agents working on the http-client Go library.

## Project Overview

HTTP client library for Go with interceptor/middleware support.

- **Module**: `github.com/germangorelkin/http-client`
- **Package**: `http_client`
- **Go Version**: 1.24+
- **Documentation**: See [specs/spec.md](specs/spec.md) for architecture and design details

## Build & Test Commands

```bash
# Build
go build -v .

# Run all tests
go test ./... -v

# Run single test by name
go test -run TestClient_Post -v

# Run tests matching pattern
go test -run TestClient_ -v

# Run specific subtest
go test -run TestClient_Get/out_is_struct -v

# Run with coverage
go test -cover ./...

# Run with race detector
go test -race ./...

# Tidy dependencies
go mod tidy

# Lint (uses golangci-lint in CI)
golangci-lint run
```

## Code Style Guidelines

### Import Organization

Group imports: stdlib first, then external packages after blank line.

```go
import (
    "bytes"
    "context"
    "encoding/json"
    "net/http"

    "github.com/stretchr/testify/assert"
)
```

### Naming Conventions

| Element | Convention | Example |
|---------|------------|---------|
| Package | snake_case | `http_client` |
| Exported types | PascalCase | `Client`, `ErrorResponse` |
| Exported functions | PascalCase | `NewClient`, `CheckResponse` |
| Unexported functions | camelCase | `parseURL`, `uniteInterceptors` |
| Option functions | `With*` prefix | `WithBaseURL`, `WithUserAgent` |
| Test functions | `Test*` or `Test_*` | `TestClient_Post`, `Test_New` |
| Test helpers | snake_case (private) | `test_client` |
| Constants | camelCase (private) | `userAgent` |

### Type Definitions

Use function types for patterns:

```go
type Handler func(*http.Request) (*http.Response, error)
type Interceptor func(*http.Request, Handler) (*http.Response, error)
type ClientOpt func(*Client) error
```

### Error Handling

1. **Early return pattern** - Return errors immediately:
   ```go
   if err != nil {
       return err
   }
   ```

2. **Custom error types** - Implement error interface:
   ```go
   type ErrorResponse struct {
       Response  *http.Response
       Message   string
       RequestID string
   }
   
   func (r *ErrorResponse) Error() string {
       return fmt.Sprintf("%v %v: %d %v",
           r.Response.Request.Method, r.Response.Request.URL,
           r.Response.StatusCode, r.Message)
   }
   ```

3. **Type assertions** for error inspection:
   ```go
   errRes, ok := err.(*ErrorResponse)
   ```

### Comments Style

- Doc comments with function name prefix:
  ```go
  // DumpInterceptor logs dump request and response
  func DumpInterceptor(req *http.Request, handler Handler) (*http.Response, error)
  ```

- Block comments for sections:
  ```go
  /*
  Examples of Interceptor
  */
  ```

### Testing Patterns

1. Use `httptest.NewServer` for HTTP mocking
2. Use `github.com/stretchr/testify/assert` for assertions
3. Use `t.Run()` for subtests
4. Use `t.Helper()` for utility functions

```go
func TestClient_Get(t *testing.T) {
    ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintln(w, `{"name":"Name"}`)
    }))
    defer ts.Close()

    cli := NewClient(nil)

    t.Run("out is struct", func(t *testing.T) {
        user := struct {
            Name string `json:"name"`
        }{}
        err := cli.Get(ts.URL, &user)
        assert.NoError(t, err)
        assert.Equal(t, "Name", user.Name)
    })
}
```

## Development Workflow

### Creating Changes

When implementing new features or fixes:

1. **Create a detailed plan** before coding:
   - Save plan to `.opencode/plan/YYYY-MM-DD_task_name.md`
   - Include all steps and implementation details
   - Document any assumptions made

2. **Ask clarifying questions** for any ambiguity:
   - Do not assume requirements
   - Confirm edge cases
   - Verify expected behavior

3. **Iterate with confirmation**:
   - Implement changes in small increments
   - Request confirmation after each iteration
   - Do not proceed to next step without approval

### Plan File Template

```markdown
# Task: [Brief Description]

**Date:** YYYY-MM-DD  
**Status:** Planning | In Progress | Completed | Cancelled

## Problem Statement

[Describe what needs to be done and why]

## Proposed Solution

[High-level approach]

## Detailed Steps

1. [ ] Step 1: [Description]
   - Files: `file1.go`, `file2.go`
   - Changes: [Brief description]
   
2. [ ] Step 2: [Description]
   - Files: `file3_test.go`
   - Changes: [Brief description]

3. [ ] Step 3: [Description]
   - Files: `README.md`
   - Changes: [Brief description]

## Testing Strategy

- [ ] Unit tests: [Description]
- [ ] Integration tests: [Description]
- [ ] Manual testing: [Description]

## Open Questions

1. [Question 1]?
2. [Question 2]?

## Risks and Edge Cases

- Risk 1: [Description and mitigation]
- Edge case 1: [Description and handling]

## Rollback Strategy

[How to undo this change if needed]
```

## Project Structure

```
.
├── client.go           # Main HTTP client implementation
├── interceptor.go      # Interceptor/middleware implementation
├── client_test.go      # Client tests
├── interceptor_test.go # Interceptor tests
├── go.mod              # Module definition
├── go.sum              # Dependency checksums
├── specs/
│   └── spec.md         # Architecture and design specification
├── .opencode/
│   └── plan/           # Change plans directory
└── .github/
    └── workflows/
        └── go.yml      # CI configuration
```

## CI/CD

GitHub Actions runs on push/PR to master:
- Platforms: ubuntu, macos, windows
- Go version: 1.24 (CI) / 1.24 (module)
- Steps: golangci-lint, test, build

## Key Patterns in This Codebase

### Functional Options Pattern

```go
client, err := New(nil,
    WithBaseURL("https://api.example.com"),
    WithUserAgent("my-client"),
    WithAuthorization("Bearer token"),
    WithInterceptor(DumpInterceptor))
```

### Interceptor Chain Pattern

```go
func LoggingInterceptor(req *http.Request, handler Handler) (*http.Response, error) {
    log.Printf("Request: %s %s", req.Method, req.URL)
    resp, err := handler(req)
    if err == nil {
        log.Printf("Response: %d", resp.StatusCode)
    }
    return resp, err
}
```

## Common Gotchas

1. **Response body must be closed**: `defer resp.Body.Close()`
2. **EOF on empty body**: Decode returns `io.EOF` for empty response - handle gracefully
3. **Transport wrapping**: Custom transport is wrapped in `interTransport`
4. **Headers are maps**: Use `map[string]string` for headers
5. **BaseURL resolution**: Uses `url.URL.Parse()` for relative paths

## Dependencies

| Package | Purpose |
|---------|---------|
| `github.com/stretchr/testify` | Test assertions (test only) |

## References

- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Effective Go](https://golang.org/doc/effective_go)
- [Project Specification](specs/spec.md)
