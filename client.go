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

type ErrorResponse struct {
	Response  *http.Response
	Message   string
	RequestID string
}

type Client struct {
	client        *http.Client
	BaseURL       *url.URL
	UserAgent     string
	Authorization string
}

type ClientOpt func(*Client) error

func NewClient(httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	if httpClient.Transport == nil {
		httpClient.Transport = interTransport{transport: http.DefaultTransport}
	} else {
		httpClient.Transport = interTransport{transport: httpClient.Transport}
	}

	return &Client{client: httpClient, UserAgent: userAgent}
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

func SetBaseURL(bu string) ClientOpt {
	return func(c *Client) error {
		u, err := url.Parse(bu)
		if err != nil {
			return err
		}

		c.BaseURL = u
		return nil
	}
}

func SetUserAgent(ua string) ClientOpt {
	return func(c *Client) error {
		c.UserAgent = fmt.Sprintf("%s %s", ua, c.UserAgent)
		return nil
	}
}

func SetAuthorization(token, tokenType string) ClientOpt {
	return func(c *Client) error {
		c.Authorization = fmt.Sprintf("%s %s", tokenType, token)
		return nil
	}
}

func SetInterceptor(inter Interceptor) ClientOpt {
	return func(c *Client) error {
		tr, ok := c.client.Transport.(interTransport)
		if !ok {
			return fmt.Errorf("error")
		}
		tr.interceptors = append(tr.interceptors, inter)
		tr.unitedInterceptor = uniteInterceptors(tr.interceptors)
		return nil
	}
}

func (c *Client) NewRequest(method, urlStr string, body interface{}) (*http.Request, error) {
	u, err := c.BaseURL.Parse(urlStr)
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
	if c.UserAgent != "" {
		req.Header.Set("User-Agent", c.UserAgent)
	}
	if c.Authorization != "" {
		req.Header.Set("Authorization", c.Authorization)
	}
	return req, nil
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
		err := json.Unmarshal(data, errorResponse)
		if err != nil {
			errorResponse.Message = string(data)
		}
	}

	return errorResponse
}

func (r *ErrorResponse) Error() string {
	if r.RequestID != "" {
		return fmt.Sprintf("%v %v: %d (request %q) %v",
			r.Response.Request.Method, r.Response.Request.URL, r.Response.StatusCode, r.RequestID, r.Message)
	}
	return fmt.Sprintf("%v %v: %d %v",
		r.Response.Request.Method, r.Response.Request.URL, r.Response.StatusCode, r.Message)
}
