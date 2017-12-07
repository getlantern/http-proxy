package domainfront

import (
	"net/http"

	"github.com/getlantern/proxy/filters"
)

// NewFilter constructs a new filter that rewrites requests in preparation for
// domain-fronting.
func NewFilter() filters.Filter {
	return &domainFrontFilter{}
}

type domainFrontFilter struct{}

func (f *domainFrontFilter) Apply(ctx filters.Context, req *http.Request, next filters.Next) (*http.Response, filters.Context, error) {
	req.URL.Scheme = req.Header.Get("X-DDF-Scheme")
	req.URL.Host = req.Header.Get("X-DDF-Host")
	req.Host = req.URL.Host
	return next(ctx, req)
}
