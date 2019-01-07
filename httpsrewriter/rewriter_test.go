package httpsrewriter

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/getlantern/mockconn"
	"github.com/getlantern/proxy/filters"
)

type dummyHandler struct{ req *http.Request }

func (h *dummyHandler) Apply(ctx filters.Context, req *http.Request, next filters.Next) (*http.Response, filters.Context, error) {
	h.req = req
	return next(ctx, req)
}

func TestRedirect(t *testing.T) {
	dummyAddr := "a.b.c.d:12345"
	dummy := &dummyHandler{}
	chain := filters.Join(&Rewriter{[]string{"site1.com", "site2.org"}}, dummy)

	req, _ := http.NewRequest("GET", "http://site1.com:80/abc.gz", nil)
	req.RemoteAddr = dummyAddr
	next := func(ctx filters.Context, req *http.Request) (*http.Response, filters.Context, error) {
		return nil, ctx, nil
	}
	ctx := filters.BackgroundContext()
	chain.Apply(ctx, req, next)
	assert.Equal(t, "", dummy.req.URL.Scheme, "should clear scheme")
	assert.Equal(t, "site1.com:443", dummy.req.Host, "should use port 443")

	for _, method := range []string{"GET", "HEAD", "PUT", "POST", "DELETE", "OPTIONS"} {
		req, _ = http.NewRequest(method, "http://site2.org/abc.gz", nil)
		req.RemoteAddr = dummyAddr
		chain.Apply(ctx, req, next)
		assert.Equal(t, "", dummy.req.URL.Scheme, "should clear scheme")
		assert.Equal(t, "site2.org:443", dummy.req.Host, "should use port 443")
	}

	req, _ = http.NewRequest("CONNECT", "http://site2.org/", nil)
	req.RemoteAddr = dummyAddr
	chain.Apply(ctx, req, next)
	assert.Equal(t, "http", dummy.req.URL.Scheme, "should not clear scheme")
	assert.Equal(t, "site2.org", dummy.req.Host, "should remain http")

	req, _ = http.NewRequest("GET", "http://site2.org:443/abc.gz", nil)
	req.RemoteAddr = "bad-addr"
	chain.Apply(ctx, req, next)
	assert.Equal(t, "", dummy.req.URL.Scheme, "should clear scheme")
	assert.Equal(t, "site2.org:443", dummy.req.Host, "should use port 443")

	req, _ = http.NewRequest("GET", "http://not-config-server.org/abc.gz", nil)
	req.RemoteAddr = dummyAddr
	chain.Apply(ctx, req, next)
	assert.Equal(t, "http", dummy.req.URL.Scheme, "should not rewrite to https for other sites")
	assert.Equal(t, "not-config-server.org", dummy.req.Host, "should not use port 443 for other sites")
}

func TestDialerConfigServer(t *testing.T) {
	d := &net.Dialer{}
	dial := (&Rewriter{[]string{"config.getiantem.org"}}).Dialer(d.DialContext)
	conn, err := dial(context.Background(), "tcp", "config.getiantem.org:443")
	assert.NoError(t, err)
	conn.Close()
}

func TestDialer(t *testing.T) {
	dummyDial := func(ctx context.Context, net, addr string) (net.Conn, error) {
		return mockconn.SucceedingDialer([]byte{}).Dial(net, addr)
	}
	d := (&Rewriter{[]string{"site1", "site2"}}).Dialer(dummyDial)
	c, _ := d(context.Background(), "tcp", "site1")
	_, ok := c.(*tls.Conn)
	assert.True(t, ok, "should override dialer if site is in list")
	c, _ = d(context.Background(), "tcp", "other")
	_, ok = c.(*tls.Conn)
	assert.False(t, ok, "should not override dialer for other dialers")
}
