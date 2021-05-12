package http_client

import (
	"bytes"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
)

type Handler func(*http.Request) (*http.Response, error)
type Interceptor func(*http.Request, Handler) (*http.Response, error)

var DefaultInterceptor Interceptor = func(req *http.Request, handler Handler) (*http.Response, error) {
	return handler(req)
}

type interTransport struct {
	transport         http.RoundTripper
	interceptors      []Interceptor
	unitedInterceptor Interceptor
}

func (t *interTransport) AddInterceptor(inter Interceptor) {
	t.interceptors = append(t.interceptors, inter)
	t.unitedInterceptor = uniteInterceptors(t.interceptors)
}

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
		body, _ := ioutil.ReadAll(resp.Body)
		body = bytes.ReplaceAll(body, []byte(":NaN"), []byte(":null"))

		resp.ContentLength = int64(len(body))
		resp.Body = ioutil.NopCloser(bytes.NewBuffer(body))
	}

	return resp, err
}
