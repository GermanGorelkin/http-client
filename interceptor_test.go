package http_client

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_uniteInterceptors(t *testing.T) {
	var got bytes.Buffer

	oneInter := func(req *http.Request, handler Handler) (*http.Response, error) {
		got.WriteString("oneInter before handler\n")
		res, err := handler(req)
		got.WriteString("oneInter after handler\n")
		return res, err
	}
	twoInter := func(req *http.Request, handler Handler) (*http.Response, error) {
		got.WriteString("twoInter before handler\n")
		res, err := handler(req)
		got.WriteString("twoInter after handler\n")
		return res, err
	}
	threeInter := func(req *http.Request, handler Handler) (*http.Response, error) {
		got.WriteString("threeInter before handler\n")
		res, err := handler(req)
		got.WriteString("threeInter after handler\n")
		return res, err
	}

	roundTrip := func(*http.Request) (*http.Response, error) {
		got.WriteString("roundTrip\n")
		return nil, nil
	}

	inters := []Interceptor{oneInter, twoInter, threeInter}
	unitedInterceptor := uniteInterceptors(inters)
	_, _ = unitedInterceptor(nil, roundTrip)

	var want bytes.Buffer
	want.WriteString("oneInter before handler\n")   // one
	want.WriteString("twoInter before handler\n")   // two
	want.WriteString("threeInter before handler\n") // three
	want.WriteString("roundTrip\n")
	want.WriteString("threeInter after handler\n") // three
	want.WriteString("twoInter after handler\n")   // two
	want.WriteString("oneInter after handler\n")   // one

	assert.Equal(t, got.String(), want.String())
}

func Test_DumpInterceptor(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	log.SetFlags(0)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, `{"name":"Name"}`)
	}))
	defer ts.Close()

	tr := interTransport{transport: http.DefaultTransport}
	tr.interceptors = append(tr.interceptors, DumpInterceptor)
	tr.unitedInterceptor = uniteInterceptors(tr.interceptors)

	client := http.Client{Transport: tr}

	res, err := client.Get(ts.URL)
	assert.NoError(t, err)

	got, err := io.ReadAll(res.Body)
	assert.NoError(t, err)
	res.Body.Close()

	want := `{"name":"Name"}` + "\n"
	assert.Equal(t, want, string(got))

	assert.True(t, buf.Len() > 0)
}
