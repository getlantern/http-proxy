package domainfront

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/getlantern/errors"
	"github.com/getlantern/proxy/filters"
)

// NewFilter constructs a new filter that rewrites requests in preparation for
// domain-fronting.
func NewFilter() filters.Filter {
	return &domainFrontFilter{}
}

type domainFrontFilter struct{}

func (f *domainFrontFilter) Apply(ctx filters.Context, req *http.Request, next filters.Next) (*http.Response, filters.Context, error) {
	fmt.Println(req)
	u := req.Header.Get("X-Ddf-Url")
	var parseErr error
	req.URL, parseErr = url.Parse(u)
	if parseErr != nil {
		return &http.Response{
			StatusCode: http.StatusBadRequest,
		}, ctx, errors.New("Unable to parse url %v: %v", u, parseErr)
	}
	req.Host = req.URL.Host
	return next(ctx, req)
}
