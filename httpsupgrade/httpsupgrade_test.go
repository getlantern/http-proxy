package httpsupgrade

import (
	"context"
	"net/http"
	"testing"

	"github.com/getlantern/proxy/v3/filters"
	"github.com/stretchr/testify/assert"
)

type CaptureTx struct {
	rt    http.RoundTripper
	req   *http.Request
	proto string
}

func (c *CaptureTx) RoundTrip(req *http.Request) (*http.Response, error) {
	c.req = req.Clone(context.Background())

	res, err := c.rt.RoundTrip(req)
	if res != nil {
		c.proto = res.Proto
	}

	return res, err
}

func (c *CaptureTx) takeRequest() *http.Request {
	r := c.req
	c.req = nil
	return r
}

func (c *CaptureTx) takeProto() string {
	p := c.proto
	c.proto = ""
	return p
}

func (c *CaptureTx) next(cs *filters.ConnectionState, req *http.Request) (*http.Response, *filters.ConnectionState, error) {
	c.req = req.Clone(context.Background())
	return nil, cs, nil
}

func captureRoundTripInfo(f filters.Filter) *CaptureTx {
	if up, ok := f.(*httpsUpgrade); ok {
		cap := &CaptureTx{up.httpClient.Transport, nil, ""}
		up.httpClient.Transport = cap
		return cap
	}
	return nil
}

func TestRedirect(t *testing.T) {
	up := NewHTTPSUpgrade("xyz")
	cap := captureRoundTripInfo(up)
	chain := filters.Join(up)

	req, _ := http.NewRequest("GET", "http://config.getiantem.org:80/proxies.yaml.gz", nil)
	// We look at this for setting the X-Lantern-Config-Server-Client-IP header thing,
	// although it shouldn't actually impact anything.
	req.RemoteAddr = "127.0.0.1:1234"
	cs := filters.NewConnectionState(req, nil, nil)

	chain.Apply(cs, req, cap.next)
	req = cap.takeRequest()
	assert.Equal(t, "https", req.URL.Scheme, "scheme should be HTTPS")
	assert.Equal(t, "config.getiantem.org:443", req.Host, "should use port 443")

	for _, method := range []string{"GET", "HEAD", "PUT", "POST", "DELETE", "OPTIONS"} {
		req, _ = http.NewRequest(method, "http://api.getiantem.org/abc.gz", nil)
		chain.Apply(cs, req, cap.next)
		req = cap.takeRequest()
		assert.Equal(t, "https", req.URL.Scheme, "scheme should be HTTPS")
		assert.Equal(t, "api.getiantem.org:443", req.Host, "should use port 443")
	}

	req, _ = http.NewRequest("CONNECT", "http://api.getiantem.org/", nil)
	chain.Apply(cs, req, cap.next)
	req = cap.takeRequest()
	assert.Equal(t, "http", req.URL.Scheme, "should not clear scheme")
	assert.Equal(t, "api.getiantem.org", req.Host, "should remain http (port 80)")

	req, _ = http.NewRequest("GET", "http://not-config-server.org/abc.gz", nil)
	chain.Apply(cs, req, cap.next)
	req = cap.takeRequest()
	assert.Equal(t, "http", req.URL.Scheme, "should not rewrite to https for other sites")
	assert.Equal(t, "not-config-server.org", req.Host, "should not use port 443 for other sites")
}

func TestHTTPS2(t *testing.T) {
	up := NewHTTPSUpgrade("xyz")
	cap := captureRoundTripInfo(up)
	chain := filters.Join(up)

	req, _ := http.NewRequest("GET", "http://config.getiantem.org/abc.gz", nil)
	next := func(cs *filters.ConnectionState, req *http.Request) (*http.Response, *filters.ConnectionState, error) {
		return nil, cs, nil
	}

	cs := filters.NewConnectionState(req, nil, nil)
	res, cs, err := chain.Apply(cs, req, next)
	assert.NoError(t, err)
	assert.NotNil(t, res)

	assert.Equal(t, "HTTP/2.0", cap.takeProto())

	r, _ := http.NewRequest("GET", "http://api.getiantem.org/abc.gz", nil)

	r.URL.Scheme = "http"
	r.URL.Host = "api.getiantem.org"
	r.Host = r.URL.Host
	r.RequestURI = ""

	res, cs, err = chain.Apply(cs, r, next)
	assert.NoError(t, err)
	assert.NotNil(t, res)

	assert.Equal(t, "HTTP/2.0", cap.takeProto())
}
