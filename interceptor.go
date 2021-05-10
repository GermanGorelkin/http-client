package http_client

import (
	"net/http"
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

func (t interTransport) RoundTrip(r *http.Request) (*http.Response, error) {
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
