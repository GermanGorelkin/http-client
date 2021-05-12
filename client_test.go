package http_client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestClient_Do_Status400(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer ts.Close()

	client := NewClient(nil)

	req, err := http.NewRequest("GET", ts.URL, nil)
	assert.NoError(t, err)
	_, err = client.Do(context.Background(), req, nil)
	assert.NotNil(t, err)
}

func TestCheckResponse(t *testing.T) {
	t.Run("status 200", func(t *testing.T) {
		res := &http.Response{
			StatusCode: 200,
		}
		assert.NoError(t, CheckResponse(res))
	})

	t.Run("status 400", func(t *testing.T) {
		res := &http.Response{
			StatusCode: 400,
			Body:       ioutil.NopCloser(bytes.NewBufferString("test msg")),
		}
		err := CheckResponse(res)
		assert.NotNil(t, err)
		errRes, ok := err.(*ErrorResponse)
		assert.Truef(t, ok, "%q is not *ErrorResponse", err)
		assert.Equal(t, "test msg", errRes.Message)
	})
}

func TestClient_NewRequest_POST(t *testing.T) {
	userAgent := "http-client"
	token := "token bG9sOnNlY3VyZQ"
	user := struct {
		Name string `json:"name"`
	}{
		Name: "testName",
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, userAgent, r.Header["User-Agent"][0])
		assert.Equal(t, token, r.Header["Authorization"][0])
		assert.Equal(t, "application/json", r.Header["Content-Type"][0])

		v := struct {
			Name string `json:"name"`
		}{}
		err := json.NewDecoder(r.Body).Decode(&v)
		assert.NoError(t, err)
		assert.Equal(t, user, v)
	}))
	defer ts.Close()

	client, err := New(nil,
		SetBaseURL(ts.URL),
		SetUserAgent(userAgent),
		SetAuthorization("bG9sOnNlY3VyZQ", "token"))
	assert.NoError(t, err)

	req, err := client.NewRequest("POST", "user", user)
	assert.NoError(t, err)
	assert.Equal(t, path.Join(ts.URL, "user"), path.Join(ts.URL, req.URL.Path))
	assert.Equal(t, req.Method, "POST")
	assert.NotNil(t, req.Body)

	_, err = http.DefaultClient.Do(req)
	assert.NoError(t, err)

	test_client(t, client)
}

func TestClient_NewRequest_GET(t *testing.T) {
	userAgent := "http-client"
	token := "token bG9sOnNlY3VyZQ"

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, userAgent, r.Header["User-Agent"][0])
		assert.Equal(t, token, r.Header["Authorization"][0])
	}))
	defer ts.Close()

	client, err := New(nil,
		SetBaseURL(ts.URL),
		SetUserAgent(userAgent),
		SetAuthorization("bG9sOnNlY3VyZQ", "token"))
	assert.NoError(t, err)

	req, err := client.NewRequest("GET", "user", nil)
	assert.NoError(t, err)
	assert.Equal(t, path.Join(ts.URL, "user"), path.Join(ts.URL, req.URL.Path))
	assert.Equal(t, req.Method, "GET")
	assert.Nil(t, req.Body)

	_, err = http.DefaultClient.Do(req)
	assert.NoError(t, err)

	test_client(t, client)
}

func Test_New(t *testing.T) {
	client, err := New(nil,
		SetBaseURL("https://golang.org/"),
		SetUserAgent("custome"),
		SetAuthorization("bG9sOnNlY3VyZQ", "token"),
		SetInterceptor(DefaultInterceptor))

	assert.NoError(t, err)
	assert.Equal(t, "golang.org", client.BaseURL.Host)
	assert.Equal(t, "custome", client.UserAgent)
	assert.Equal(t, "token bG9sOnNlY3VyZQ", client.Authorization)
	assert.Truef(t, len(client.client.Transport.(*interTransport).interceptors) == 1,
		"len=%d", len(client.client.Transport.(*interTransport).interceptors))

	test_client(t, client)
}

func Test_NewClient(t *testing.T) {
	t.Run("httpClient is nil", func(t *testing.T) {
		client := NewClient(nil)

		test_client(t, client)

		assert.NotNil(t, client.client.Transport)

		tr, ok := client.client.Transport.(*interTransport)
		assert.Truef(t, ok, "Transport is not interTransport")

		assert.Equal(t, http.DefaultTransport, tr.transport)
	})
	t.Run("custom client and transport", func(t *testing.T) {
		custTr := &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   10 * time.Second,
				KeepAlive: 20 * time.Second,
			}).DialContext,
			MaxIdleConnsPerHost: 100,
		}
		custClient := &http.Client{
			Transport: custTr,
			Timeout:   5 * time.Second,
		}

		client := NewClient(custClient)

		test_client(t, client)

		assert.Equal(t, custClient, client.client)
		assert.NotNil(t, client.client.Transport)

		tr, ok := client.client.Transport.(*interTransport)
		assert.Truef(t, ok, "Transport is not interTransport")

		assert.Equal(t, custTr, tr.transport)
	})
}

func TestClient_AddInterceptor(t *testing.T) {
	client := NewClient(nil)

	var got string
	err := client.AddInterceptor(func(req *http.Request, handler Handler) (*http.Response, error) {
		got = "AddInterceptor"
		return handler(req)
	})
	assert.NoError(t, err)

	test_client(t, client)

	assert.Equal(t, "AddInterceptor", got)
}

func test_client(t *testing.T, client *Client) {
	t.Helper()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, `{"name":"Name"}`)
	}))
	defer ts.Close()

	res, err := client.client.Get(ts.URL)
	assert.NoError(t, err)

	got, err := io.ReadAll(res.Body)
	assert.NoError(t, err)
	res.Body.Close()

	want := `{"name":"Name"}` + "\n"
	assert.Equal(t, want, string(got))
}
