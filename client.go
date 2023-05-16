package http_client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
)

const (
	userAgent = "http-client"
)

func Get(url string, out interface{}) error {
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

func Post(url string, in, out interface{}) error {
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

type Client struct {
	client  *http.Client
	BaseURL *url.URL
	Headers map[string]string
}

type ClientOpt func(*Client) error

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

func New(httpClient *http.Client, opts ...ClientOpt) (*Client, error) {
	c := NewClient(httpClient)
	for _, opt := range opts {
		if err := opt(c); err != nil {
			return nil, err
		}
	}

	return c, nil
}

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

func WithUserAgent(ua string) ClientOpt {
	return func(c *Client) error {
		c.Headers["User-Agent"] = ua
		return nil
	}
}

func WithAuthorization(auth string) ClientOpt {
	return func(c *Client) error {
		c.Headers["Authorization"] = auth
		return nil
	}
}

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

func (c *Client) SetAuthorization(auth string) {
	c.Headers["Authorization"] = auth
}

func (c *Client) SetHeader(key, value string) {
	c.Headers[key] = value
}

func (c *Client) AddInterceptor(inter Interceptor) error {
	tr, ok := c.client.Transport.(*interTransport)
	if !ok {
		return fmt.Errorf("error")
	}
	tr.AddInterceptor(inter)
	return nil
}

func (c *Client) Get(url string, out interface{}) error {
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

func (c *Client) Post(url string, in, out interface{}) error {
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

func (c *Client) NewRequest(method, urlStr string, body interface{}) (*http.Request, error) {
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

func (c *Client) parseURL(urlStr string) (*url.URL, error) {
	if c.BaseURL == nil {
		return url.ParseRequestURI(urlStr)
	}
	return c.BaseURL.Parse(urlStr)
}

func (c *Client) Do(ctx context.Context, req *http.Request, v interface{}) (*http.Response, error) {
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

func DoRequestWithClient(ctx context.Context, client *http.Client, req *http.Request) (*http.Response, error) {
	req = req.WithContext(ctx)
	return client.Do(req)
}

func CheckResponse(r *http.Response) error {
	if c := r.StatusCode; c >= 200 && c <= 299 {
		return nil
	}

	errorResponse := &ErrorResponse{Response: r}
	data, err := ioutil.ReadAll(r.Body)
	if err == nil && len(data) > 0 {
		errorResponse.Message = string(data)
	}
	return errorResponse
}

type ErrorResponse struct {
	Response  *http.Response
	Message   string
	RequestID string
}

func (r *ErrorResponse) Error() string {
	if r.RequestID != "" {
		return fmt.Sprintf("%v %v: %d (request %q) %v",
			r.Response.Request.Method, r.Response.Request.URL, r.Response.StatusCode, r.RequestID, r.Message)
	}
	return fmt.Sprintf("%v %v: %d %v",
		r.Response.Request.Method, r.Response.Request.URL, r.Response.StatusCode, r.Message)
}
