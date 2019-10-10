package httpsrewriter

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/getlantern/proxy/filters"
)

type dummyHandler struct{ req *http.Request }

func (h *dummyHandler) Apply(ctx filters.Context, req *http.Request, next filters.Next) (*http.Response, filters.Context, error) {
	h.req = req
	return next(ctx, req)
}

func TestRedirect(t *testing.T) {
	chain := filters.Join(NewRewriter())

	req, _ := http.NewRequest("GET", "http://config.getiantem.org:80/abc.gz", nil)
	next := func(ctx filters.Context, req *http.Request) (*http.Response, filters.Context, error) {
		return nil, ctx, nil
	}
	ctx := filters.BackgroundContext()

	chain.Apply(ctx, req, next)

	assert.Equal(t, "https", req.URL.Scheme, "scheme should be HTTPS")
	assert.Equal(t, "config.getiantem.org:443", req.Host, "should use port 443")

	for _, method := range []string{"GET", "HEAD", "PUT", "POST", "DELETE", "OPTIONS"} {
		req, _ = http.NewRequest(method, "http://api.getiantem.org/abc.gz", nil)
		chain.Apply(ctx, req, next)
		assert.Equal(t, "https", req.URL.Scheme, "scheme should be HTTPS")
		assert.Equal(t, "api.getiantem.org:443", req.Host, "should use port 443")
	}

	req, _ = http.NewRequest("CONNECT", "http://api.getiantem.org/", nil)
	chain.Apply(ctx, req, next)
	assert.Equal(t, "http", req.URL.Scheme, "should not clear scheme")
	assert.Equal(t, "api.getiantem.org", req.Host, "should remain http (port 80)")

	req, _ = http.NewRequest("GET", "http://not-config-server.org/abc.gz", nil)
	chain.Apply(ctx, req, next)
	assert.Equal(t, "http", req.URL.Scheme, "should not rewrite to https for other sites")
	assert.Equal(t, "not-config-server.org", req.Host, "should not use port 443 for other sites")
}
