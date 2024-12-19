package proxyfilters

import (
	"net"
	"net/http"

	"github.com/getlantern/proxy/v3/filters"
)

const (
	xForwardedFor = "X-Forwarded-For"
)

// AddForwardedFor adds an X-Forwarded-For header based on the request's
// RemoteAddr.
var AddForwardedFor = filters.FilterFunc(func(cs *filters.ConnectionState, req *http.Request, next filters.Next) (*http.Response, *filters.ConnectionState, error) {
	if req.Method != http.MethodConnect {
		if clientIP, _, err := net.SplitHostPort(req.RemoteAddr); err == nil {
			req.Header.Add(xForwardedFor, clientIP)
		}
	}
	return next(cs, req)
})
