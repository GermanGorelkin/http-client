package http_client

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

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
