package http_client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
)

const (
	userAgent = "http-client"
)

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

// NewMultipartForm creates an empty multipart form
func NewMultipartForm() *MultipartForm {
	return &MultipartForm{
		fields: make(map[string][]string),
		files:  make([]multipartFile, 0),
	}
}

// AddField adds a text field to the form (can be called multiple times for same name)
// Note: name and value should not be empty strings.
func (m *MultipartForm) AddField(name, value string) {
	m.fields[name] = append(m.fields[name], value)
}

// AddFile adds a file to the form from an io.Reader.
// Note: fieldName and fileName should not be empty strings, and reader should not be nil.
// The entire file content will be buffered in memory when building the multipart body.
func (m *MultipartForm) AddFile(fieldName, fileName string, reader io.Reader) {
	m.files = append(m.files, multipartFile{
		fieldName: fieldName,
		fileName:  fileName,
		reader:    reader,
	})
}

// buildMultipartBody creates multipart body and returns content type with boundary.
// Note: This method buffers the entire form data in memory. For large files,
// consider using streaming alternatives or breaking uploads into chunks.
// The returned io.Reader is backed by an in-memory buffer.
func (m *MultipartForm) buildMultipartBody() (io.Reader, string, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add fields
	for name, values := range m.fields {
		for _, value := range values {
			if err := writer.WriteField(name, value); err != nil {
				return nil, "", err
			}
		}
	}

	// Add files
	for _, file := range m.files {
		part, err := writer.CreateFormFile(file.fieldName, file.fileName)
		if err != nil {
			return nil, "", err
		}
		if _, err := io.Copy(part, file.reader); err != nil {
			return nil, "", err
		}
	}

	// Close writer to finalize boundary
	if err := writer.Close(); err != nil {
		return nil, "", err
	}

	return &buf, writer.FormDataContentType(), nil
}

// Get sends a GET request to the specified URL and decodes the response into out.
func Get(url string, out any) error {
	c := NewClient(nil)
	req, err := c.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	_, err = c.Do(context.Background(), req, out)
	if err != nil {
		return err
	}
	return nil
}

// Post sends a POST request to the specified URL with in as the request body and decodes the response into out.
func Post(url string, in, out any) error {
	c := NewClient(nil)
	req, err := c.NewRequest("POST", url, in)
	if err != nil {
		return err
	}
	_, err = c.Do(context.Background(), req, out)
	if err != nil {
		return err
	}
	return nil
}

// Client is an HTTP client with support for interceptors and middleware.
type Client struct {
	client  *http.Client
	BaseURL *url.URL
	Headers map[string]string
}

// ClientOpt is a functional option for configuring a Client.
type ClientOpt func(*Client) error

// NewClient creates a new Client with the specified http.Client.
// If httpClient is nil, a default http.Client is used.
func NewClient(httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{}
	}
	if httpClient.Transport == nil {
		httpClient.Transport = &interTransport{transport: http.DefaultTransport}
	} else {
		httpClient.Transport = &interTransport{transport: httpClient.Transport}
	}

	cl := &Client{
		client:  httpClient,
		Headers: map[string]string{},
	}
	cl.Headers["User-Agent"] = userAgent

	return cl
}

// New creates a new Client with the specified http.Client and applies the given options.
func New(httpClient *http.Client, opts ...ClientOpt) (*Client, error) {
	c := NewClient(httpClient)
	for _, opt := range opts {
		if err := opt(c); err != nil {
			return nil, err
		}
	}

	return c, nil
}

// WithBaseURL sets the base URL for the Client.
func WithBaseURL(bu string) ClientOpt {
	return func(c *Client) error {
		u, err := url.Parse(bu)
		if err != nil {
			return err
		}

		c.BaseURL = u
		return nil
	}
}

// WithUserAgent sets the User-Agent header for the Client.
func WithUserAgent(ua string) ClientOpt {
	return func(c *Client) error {
		c.Headers["User-Agent"] = ua
		return nil
	}
}

// WithAuthorization sets the Authorization header for the Client.
func WithAuthorization(auth string) ClientOpt {
	return func(c *Client) error {
		c.Headers["Authorization"] = auth
		return nil
	}
}

// WithInterceptor adds an interceptor to the Client.
func WithInterceptor(inter Interceptor) ClientOpt {
	return func(c *Client) error {
		tr, ok := c.client.Transport.(*interTransport)
		if !ok {
			return fmt.Errorf("error")
		}
		tr.AddInterceptor(inter)
		return nil
	}
}

// SetAuthorization sets the Authorization header for the Client.
func (c *Client) SetAuthorization(auth string) {
	c.Headers["Authorization"] = auth
}

// SetHeader sets a header for the Client.
func (c *Client) SetHeader(key, value string) {
	c.Headers[key] = value
}

// AddInterceptor adds an interceptor to the Client.
func (c *Client) AddInterceptor(inter Interceptor) error {
	tr, ok := c.client.Transport.(*interTransport)
	if !ok {
		return fmt.Errorf("error")
	}
	tr.AddInterceptor(inter)
	return nil
}

// Get sends a GET request to the specified URL and decodes the response into out.
func (c *Client) Get(url string, out any) error {
	req, err := c.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	_, err = c.Do(context.Background(), req, out)
	if err != nil {
		return err
	}
	return nil
}

// Post sends a POST request to the specified URL with in as the request body and decodes the response into out.
func (c *Client) Post(url string, in, out any) error {
	req, err := c.NewRequest("POST", url, in)
	if err != nil {
		return err
	}
	_, err = c.Do(context.Background(), req, out)
	if err != nil {
		return err
	}
	return nil
}

// PostMultipart sends a POST multipart request and decodes the response into out.
func (c *Client) PostMultipart(url string, form *MultipartForm, out any) error {
	req, err := c.NewMultipartRequest("POST", url, form)
	if err != nil {
		return err
	}
	_, err = c.Do(context.Background(), req, out)
	if err != nil {
		return err
	}
	return nil
}

// NewRequest creates a new HTTP request with the specified method, URL, and body.
func (c *Client) NewRequest(method, urlStr string, body any) (*http.Request, error) {
	u, err := c.parseURL(urlStr)
	if err != nil {
		return nil, err
	}

	var buf io.ReadWriter
	if body != nil {
		buf = &bytes.Buffer{}
		err = json.NewEncoder(buf).Encode(body)
		if err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequest(method, u.String(), buf)
	if err != nil {
		return nil, err
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range c.Headers {
		req.Header.Set(k, v)
	}

	return req, nil
}

// NewMultipartRequest creates a new multipart HTTP request with the specified form data.
func (c *Client) NewMultipartRequest(method, urlStr string, form *MultipartForm) (*http.Request, error) {
	u, err := c.parseURL(urlStr)
	if err != nil {
		return nil, err
	}

	body, contentType, err := form.buildMultipartBody()
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(method, u.String(), body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", contentType)
	for k, v := range c.Headers {
		req.Header.Set(k, v)
	}

	return req, nil
}

func (c *Client) parseURL(urlStr string) (*url.URL, error) {
	if c.BaseURL == nil {
		return url.ParseRequestURI(urlStr)
	}
	return c.BaseURL.Parse(urlStr)
}

// Do sends an HTTP request and decodes the response into v.
func (c *Client) Do(ctx context.Context, req *http.Request, v any) (*http.Response, error) {
	resp, err := DoRequestWithClient(ctx, c.client, req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err = CheckResponse(resp); err != nil {
		return resp, err
	}

	switch v := v.(type) {
	case nil:
	case io.Writer:
		_, err = io.Copy(v, resp.Body)
	default:
		decErr := json.NewDecoder(resp.Body).Decode(v)
		if decErr == io.EOF {
			decErr = nil // ignore EOF errors caused by empty response body
		}
		if decErr != nil {
			err = decErr
		}
	}

	return resp, err
}

// DoRequestWithClient sends an HTTP request using the specified client.
func DoRequestWithClient(ctx context.Context, client *http.Client, req *http.Request) (*http.Response, error) {
	req = req.WithContext(ctx)
	return client.Do(req)
}

// CheckResponse checks if the HTTP response indicates an error.
func CheckResponse(r *http.Response) error {
	if c := r.StatusCode; c >= 200 && c <= 299 {
		return nil
	}

	errorResponse := &ErrorResponse{Response: r}
	data, err := io.ReadAll(r.Body)
	if err == nil && len(data) > 0 {
		errorResponse.Message = string(data)
	}
	return errorResponse
}

// ErrorResponse represents an error response from an HTTP request.
type ErrorResponse struct {
	Response  *http.Response
	Message   string
	RequestID string
}

// Error returns a string representation of the error.
func (r *ErrorResponse) Error() string {
	if r.RequestID != "" {
		return fmt.Sprintf("%v %v: %d (request %q) %v",
			r.Response.Request.Method, r.Response.Request.URL, r.Response.StatusCode, r.RequestID, r.Message)
	}
	return fmt.Sprintf("%v %v: %d %v",
		r.Response.Request.Method, r.Response.Request.URL, r.Response.StatusCode, r.Message)
}
