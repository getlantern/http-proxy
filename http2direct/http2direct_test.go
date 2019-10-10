package http2direct

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/getlantern/http-proxy-lantern/httpsrewriter"
	"github.com/getlantern/proxy/filters"
)

func TestHTTPS2(t *testing.T) {
	chain := filters.Join(httpsrewriter.NewRewriter(), NewHTTP2Direct())
	req, _ := http.NewRequest("GET", "http://config.getiantem.org/abc.gz", nil)
	next := func(ctx filters.Context, req *http.Request) (*http.Response, filters.Context, error) {
		return nil, ctx, nil
	}
	ctx := filters.BackgroundContext()

	res, ctx, err := chain.Apply(ctx, req, next)
	assert.NoError(t, err)
	assert.NotNil(t, res)

	assert.Equal(t, "HTTP/2.0", res.Proto)

	r, _ := http.NewRequest("GET", "http://api.getiantem.org/abc.gz", nil)

	r.URL.Scheme = "http"
	r.URL.Host = "api.getiantem.org"
	r.Host = r.URL.Host
	r.RequestURI = ""

	res, ctx, err = chain.Apply(ctx, r, next)
	assert.NoError(t, err)
	assert.NotNil(t, res)

	assert.Equal(t, "HTTP/2.0", res.Proto)
}
