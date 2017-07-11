// +build !linux

package bbr

import (
	"context"
	"net"
	"net/http"

	"github.com/getlantern/proxy/filters"
)

// noopMiddleware is used on non-linux platforms where BBR is unavailable.
type noopMiddleware struct{}

func New() Middleware {
	return &noopMiddleware{}
}

func (bm *noopMiddleware) Apply(ctx context.Context, req *http.Request, next filters.Next) (*http.Response, context.Context, error) {
	return next(ctx, req)
}

func (bm *noopMiddleware) AddMetrics(ctx context.Context, req *http.Request, resp *http.Response) {
}

func (bm *noopMiddleware) Wrap(l net.Listener) net.Listener {
	return l
}

func (bm *noopMiddleware) ABE(ctx context.Context) float64 {
	return 0
}
