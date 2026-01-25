package http_client

import (
	"bytes"
	"io"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/http/httputil"
	"time"
)

// Handler is a function that processes an HTTP request and returns a response.
type Handler func(*http.Request) (*http.Response, error)

// Interceptor is a middleware function that can intercept and modify HTTP requests and responses.
type Interceptor func(*http.Request, Handler) (*http.Response, error)

// DefaultInterceptor is a no-op interceptor that simply passes the request to the handler.
var DefaultInterceptor Interceptor = func(req *http.Request, handler Handler) (*http.Response, error) {
	return handler(req)
}

type interTransport struct {
	transport         http.RoundTripper
	interceptors      []Interceptor
	unitedInterceptor Interceptor
}

// AddInterceptor adds an interceptor to the transport.
func (t *interTransport) AddInterceptor(inter Interceptor) {
	t.interceptors = append(t.interceptors, inter)
	t.unitedInterceptor = uniteInterceptors(t.interceptors)
}

// RoundTrip executes a single HTTP transaction, applying interceptors if present.
func (t *interTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	if t.unitedInterceptor == nil {
		return t.transport.RoundTrip(r)
	}
	return t.unitedInterceptor(r, t.transport.RoundTrip)
}

func uniteInterceptors(interceptors []Interceptor) Interceptor {
	if len(interceptors) == 0 {
		return DefaultInterceptor
	}

	return func(req *http.Request, handler Handler) (*http.Response, error) {
		tailhandler := func(innerReq *http.Request) (*http.Response, error) {
			unitedInterceptor := uniteInterceptors(interceptors[1:])
			return unitedInterceptor(req, handler)
		}
		headInterceptor := interceptors[0]
		return headInterceptor(req, tailhandler)
	}
}

/*
Examples of Interceptor
*/

// DumpInterceptor logs dump request and response
func DumpInterceptor(req *http.Request, handler Handler) (*http.Response, error) {
	if bytes, err := httputil.DumpRequestOut(req, true); err == nil {
		log.Printf("%q", bytes)
	}
	resp, err := handler(req)
	if err == nil {
		if bytes, dumpError := httputil.DumpResponse(resp, true); dumpError == nil {
			log.Printf("%q", bytes)
		}
	}

	return resp, err
}

// ResponseInterceptor replaces 'NaN' with 'null' in Response.Body
// {"name":NaN} - incorrect json
// to
// {"name":null} - correct json
func ResponseInterceptor(req *http.Request, handler Handler) (*http.Response, error) {
	resp, err := handler(req)
	if err == nil {
		body, _ := io.ReadAll(resp.Body)
		body = bytes.ReplaceAll(body, []byte(":NaN"), []byte(":null"))

		resp.ContentLength = int64(len(body))
		resp.Body = io.NopCloser(bytes.NewBuffer(body))
	}

	return resp, err
}

// RetryConfig holds configuration for retry behavior
type RetryConfig struct {
	MaxRetries    int
	BaseDelay     time.Duration
	MaxDelay      time.Duration
	RetryOnStatus map[int]bool
	RetryMethods  map[string]bool
	RetryOnError  func(error) bool
}

// RetryOpt is a functional option for configuring RetryInterceptor
type RetryOpt func(*RetryConfig)

// WithMaxRetries sets the maximum number of retry attempts
func WithMaxRetries(n int) RetryOpt {
	return func(c *RetryConfig) {
		c.MaxRetries = n
	}
}

// WithBackoff sets the base and maximum delay for exponential backoff
func WithBackoff(base, max time.Duration) RetryOpt {
	return func(c *RetryConfig) {
		c.BaseDelay = base
		c.MaxDelay = max
	}
}

// WithRetryOnStatus sets which HTTP status codes should trigger a retry
func WithRetryOnStatus(codes []int) RetryOpt {
	return func(c *RetryConfig) {
		c.RetryOnStatus = make(map[int]bool)
		for _, code := range codes {
			c.RetryOnStatus[code] = true
		}
	}
}

// WithRetryMethods sets which HTTP methods are safe to retry
func WithRetryMethods(methods []string) RetryOpt {
	return func(c *RetryConfig) {
		c.RetryMethods = make(map[string]bool)
		for _, method := range methods {
			c.RetryMethods[method] = true
		}
	}
}

// WithRetryOnError sets a custom function to determine if an error should trigger a retry
func WithRetryOnError(fn func(error) bool) RetryOpt {
	return func(c *RetryConfig) {
		c.RetryOnError = fn
	}
}

// DefaultRetryInterceptor returns a retry interceptor with sensible defaults
func DefaultRetryInterceptor() Interceptor {
	return NewRetryInterceptor()
}

// NewRetryInterceptor creates a configurable retry interceptor
func NewRetryInterceptor(opts ...RetryOpt) Interceptor {
	config := &RetryConfig{
		MaxRetries: 3,
		BaseDelay:  100 * time.Millisecond,
		MaxDelay:   1 * time.Second,
		RetryOnStatus: map[int]bool{
			408: true, // Request Timeout
			429: true, // Too Many Requests
			500: true, // Internal Server Error
			502: true, // Bad Gateway
			503: true, // Service Unavailable
			504: true, // Gateway Timeout
		},
		RetryMethods: map[string]bool{
			"GET":     true,
			"HEAD":    true,
			"PUT":     true,
			"DELETE":  true,
			"OPTIONS": true,
		},
		RetryOnError: func(err error) bool {
			// Retry on timeout errors (Temporary() is deprecated since Go 1.18)
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				return true
			}
			return false
		},
	}

	for _, opt := range opts {
		opt(config)
	}

	return func(req *http.Request, handler Handler) (*http.Response, error) {
		// Check if method is safe to retry
		if !config.RetryMethods[req.Method] {
			return handler(req)
		}

		// Buffer request body once before first attempt to preserve it for retries
		var bodyBytes []byte
		var bodyErr error
		if req.Body != nil && req.Body != http.NoBody {
			bodyBytes, bodyErr = io.ReadAll(req.Body)
			if bodyErr != nil {
				return nil, bodyErr
			}
			// Reset original request body for first attempt
			req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}

		var lastErr error
		var lastResp *http.Response

		for attempt := 0; attempt <= config.MaxRetries; attempt++ {
			// For retry attempts (except first), create a new request with buffered body
			if attempt > 0 {
				clonedReq := req.Clone(req.Context())
				if len(bodyBytes) > 0 {
					clonedReq.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
				}
				req = clonedReq
			}

			resp, err := handler(req)
			lastResp = resp
			lastErr = err

			shouldRetry := false

			// Check if we should retry based on error
			if err != nil && config.RetryOnError != nil && config.RetryOnError(err) {
				shouldRetry = true
			} else if err != nil {
				// Non-retryable error
				return resp, err
			}

			// Check if we should retry based on status code
			if err == nil && resp != nil && config.RetryOnStatus[resp.StatusCode] {
				shouldRetry = true
			}

			// If we shouldn't retry, return the result
			if !shouldRetry {
				return resp, err
			}

			// If this was the last attempt, break without closing
			// (caller will handle closing the final response)
			if attempt == config.MaxRetries {
				break
			}

			// Close response body before retrying to prevent leaks
			if resp != nil && resp.Body != nil {
				resp.Body.Close()
			}

			if req.Context().Err() != nil {
				return lastResp, lastErr
			}

			// Calculate backoff with jitter
			delay := calculateBackoff(config.BaseDelay, config.MaxDelay, attempt)
			time.Sleep(delay)
		}

		// Return last response/error after all retries exhausted
		if lastErr != nil {
			return lastResp, lastErr
		}
		return lastResp, nil
	}
}

// calculateBackoff calculates exponential backoff with jitter
func calculateBackoff(baseDelay, maxDelay time.Duration, attempt int) time.Duration {
	// Exponential backoff: base * 2^attempt
	delay := baseDelay * time.Duration(1<<uint(attempt))

	// Cap at max delay
	if delay > maxDelay {
		delay = maxDelay
	}

	// Add jitter: ±15%
	jitter := 0.15
	jitterFactor := 1.0 + (rand.Float64()*2*jitter - jitter)
	delay = time.Duration(float64(delay) * jitterFactor)

	return delay
}
