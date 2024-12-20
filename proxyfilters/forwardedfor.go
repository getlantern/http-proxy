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
			// Proxies are supposed to actually overwrite previous values, as they
			// can be maliciously set by the client.
			req.Header.Set(xForwardedFor, clientIP)
		} else {
			// If we can't parse the client IP, we should remove the header.
			req.Header.Del(xForwardedFor)
		}
	}
	return next(cs, req)
})
