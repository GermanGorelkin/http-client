package http_client

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_NewClient(t *testing.T) {
	t.Run("httpClient is nil", func(t *testing.T) {
		client := NewClient(nil)

		test_client(t, client)

		assert.Equal(t, http.DefaultClient, client.client)
		assert.NotNil(t, client.client.Transport)

		tr, ok := client.client.Transport.(interTransport)
		assert.Truef(t, ok, "Transport is not interTransport")

		assert.Equal(t, http.DefaultTransport, tr.transport)
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
