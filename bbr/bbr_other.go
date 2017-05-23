// +build !linux

package bbr

import (
	"net"
	"net/http"

	"github.com/getlantern/http-proxy/filters"
)

// noopMiddleware is used on non-linux platforms where BBR is unavailable.
type noopMiddleware struct{}

func New() Middleware {
	return &noopMiddleware{}
}

func (bm *noopMiddleware) Apply(w http.ResponseWriter, req *http.Request, next filters.Next) error {
	return next()
}

func (bm *noopMiddleware) AddMetrics(resp *http.Response) *http.Response {
	return resp
}

func (bm *noopMiddleware) Wrap(l net.Listener) net.Listener {
	return l
}

func (bm *noopMiddleware) ABE(req *http.Request) float64 {
	return 0
}
