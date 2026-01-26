# Task: Add multipart file upload support

**Date:** 2026-01-26  
**Status:** Completed

## Problem Statement

The library currently supports JSON request bodies only. We need to add multipart/form-data file upload capability while preserving existing JSON behavior and interceptor flow.

## Proposed Solution

Introduce a multipart request builder that can attach files and form fields via `io.Reader`, and expose a high-level API on `Client` to create and send multipart requests. Ensure `Content-Type` is set correctly with boundary, and keep existing JSON request creation unchanged.

## API Design

### New Types

```go
// MultipartForm holds multipart form data
type MultipartForm struct {
    fields map[string][]string
    files  []multipartFile
}

type multipartFile struct {
    fieldName string
    fileName  string
    reader    io.Reader
}
```

### New Functions and Methods

```go
// NewMultipartForm creates an empty multipart form
func NewMultipartForm() *MultipartForm

// AddField adds a text field to the form (can be called multiple times for same name)
func (m *MultipartForm) AddField(name, value string)

// AddFile adds a file to the form from an io.Reader
func (m *MultipartForm) AddFile(fieldName, fileName string, reader io.Reader)

// NewMultipartRequest creates a new multipart HTTP request
func (c *Client) NewMultipartRequest(method, urlStr string, form *MultipartForm) (*http.Request, error)

// PostMultipart sends a POST multipart request and decodes the response into out
func (c *Client) PostMultipart(url string, form *MultipartForm, out any) error
```

### Usage Example

```go
form := NewMultipartForm()
form.AddField("description", "my file")
form.AddFile("attachment", "photo.jpg", fileReader)

var resp Response
err := client.PostMultipart("/upload", form, &resp)
```

## Detailed Steps

1. [x] Implement multipart types and body construction
   - Files: `client.go`
   - Changes:
     - Add `MultipartForm` struct and `multipartFile` internal struct.
     - Add `NewMultipartForm()` constructor.
     - Add `AddField(name, value string)` method.
     - Add `AddFile(fieldName, fileName string, reader io.Reader)` method.
     - Use `mime/multipart` to write fields and files to buffer.
     - Internal method to build body and return `Content-Type` with boundary.

2. [x] Implement Client methods for multipart requests
   - Files: `client.go`
   - Changes:
     - Add `NewMultipartRequest(method, urlStr string, form *MultipartForm) (*http.Request, error)`.
     - Add `PostMultipart(url string, form *MultipartForm, out any) error`.
     - Ensure boundary-based `Content-Type` is set on request.
     - Preserve default headers, merge with multipart content type.
   - Dependencies: Step 1

3. [x] Add tests for multipart upload
   - Files: `client_test.go`
   - Changes:
     - Use `httptest.NewServer` to verify form fields and file contents.
     - Add subtests for:
       - single file + fields
       - multiple files under same field name
       - only fields (no files)
       - only files (no fields)
     - Assert `Content-Type` starts with `multipart/form-data; boundary=`.
     - Verify file content matches original reader content.
   - Dependencies: Steps 1-2

4. [x] Update documentation
   - Files: `README.md`, `specs/spec.md`
   - Changes:
     - Add usage example for multipart upload.
     - Document `MultipartForm` API.
     - Add `PostMultipart` and `NewMultipartRequest` to API reference.
   - Dependencies: Steps 1-3 (after API is stable)

## Testing Strategy

- [x] Unit tests: `TestClient_PostMultipart_*` in `client_test.go`
  - `TestClient_PostMultipart_SingleFileWithFields`
  - `TestClient_PostMultipart_MultipleFiles`
  - `TestClient_PostMultipart_OnlyFields`
  - `TestClient_PostMultipart_OnlyFiles`
- [x] Integration tests: validate multipart parsing with `r.ParseMultipartForm`.
- [x] Manual testing: optional sample upload to local server.

## Decisions Made

1. **File source**: Only `io.Reader` is supported (no file path helpers).
2. **JSON `NewRequest`**: Remains unchanged. Multipart uses separate `NewMultipartRequest`.
3. **`PostMultipart` method**: Will be added as convenience method.

## Risks and Edge Cases

- **Risk 1**: Large file buffering (multipart writer uses in-memory buffer).
  - Mitigation: Acceptable for first iteration. TODO: consider streaming with `io.Pipe` in future.
- **Edge case 1**: Multiple files with same form field name.
  - Handling: `AddFile` can be called multiple times with same `fieldName`.
- **Edge case 2**: Empty form (no fields, no files).
  - Handling: Return valid empty multipart body.
- **Edge case 3**: Reader returns error during copy.
  - Handling: Propagate error from `NewMultipartRequest`.

## Rollback Strategy

Revert new multipart methods and struct, and remove corresponding tests and documentation changes. No changes to existing `NewRequest` or `Post` methods.