package cleanheadersfilter

import (
	"net/http"
	"strings"

	"github.com/getlantern/proxy/filters"

	"github.com/getlantern/http-proxy-lantern/domains"
)

type filter struct {
}

func New() filters.Filter {
	return &filter{}
}

func (f *filter) Apply(ctx filters.Context, req *http.Request, next filters.Next) (*http.Response, filters.Context, error) {
	f.stripHeadersIfNecessary(req)
	return next(ctx, req)
}

func (f *filter) stripHeadersIfNecessary(req *http.Request) {
	if !domains.ConfigForRequest(req).PassInternalHeaders {
		for header := range req.Header {
			if strings.HasPrefix(header, "X-Lantern") {
				req.Header.Del(header)
			}
		}
	}
}
