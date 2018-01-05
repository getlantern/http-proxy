package configserverfilter

import (
	"crypto/tls"
	"net"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/getlantern/http-proxy-lantern/common"
	"github.com/getlantern/mockconn"
	"github.com/getlantern/proxy/filters"
)

type dummyHandler struct{ req *http.Request }

func (h *dummyHandler) Apply(ctx filters.Context, req *http.Request, next filters.Next) (*http.Response, filters.Context, error) {
	h.req = req
	return next(ctx, req)
}

func TestDomainCache(t *testing.T) {
	csf := New(&Options{"fake-token", []string{"site1.com", "site2.org"}})
	ip := csf.resolveDomain("config-do-roundrobin.getiantem.org")
	//log.Debugf("IP is: %v", ip)
	assert.True(t, len(ip) > 0)

	ip = csf.resolveDomain("somethingtotallybogus49823842304819084019.org")
	//log.Debugf("IP is: %v", ip)
	assert.True(t, len(ip) > 0)
}

func TestModifyRequest(t *testing.T) {
	fakeToken := "fake-token"
	dummyClientIP := "1.1.1.1"
	dummyAddr := dummyClientIP + ":12345"
	dummy := &dummyHandler{}
	csf := New(&Options{fakeToken, []string{"site1.com", "site2.org"}})
	chain := filters.Join(csf, dummy)

	req, _ := http.NewRequest("GET", "http://site1.com:80/abc.gz", nil)
	req.RemoteAddr = dummyAddr
	next := func(ctx filters.Context, req *http.Request) (*http.Response, filters.Context, error) {
		return nil, ctx, nil
	}
	ctx := filters.BackgroundContext()
	chain.Apply(ctx, req, next)
	assert.Equal(t, "https", dummy.req.URL.Scheme, "should rewrite to https")
	assert.Equal(t, csf.dnsCache["site1.com"]+":443", dummy.req.Host, "should use port 443")
	assert.Equal(t, fakeToken, dummy.req.Header.Get(common.CfgSvrAuthTokenHeader), "should attach token")
	assert.Equal(t, dummyClientIP, dummy.req.Header.Get(common.CfgSvrClientIPHeader), "should attach client ip")

	req, _ = http.NewRequest("GET", "http://site2.org/abc.gz", nil)
	req.RemoteAddr = dummyAddr
	chain.Apply(ctx, req, next)
	assert.Equal(t, "https", dummy.req.URL.Scheme, "should rewrite to https")
	assert.Equal(t, csf.dnsCache["site2.org"]+":443", dummy.req.Host, "should use port 443")
	assert.Equal(t, fakeToken, dummy.req.Header.Get(common.CfgSvrAuthTokenHeader), "should attach token")
	assert.Equal(t, dummyClientIP, dummy.req.Header.Get(common.CfgSvrClientIPHeader), "should attach client ip")

	req, _ = http.NewRequest("GET", "http://site2.org:443/abc.gz", nil)
	req.RemoteAddr = "bad-addr"
	chain.Apply(ctx, req, next)
	assert.Equal(t, "https", dummy.req.URL.Scheme, "should rewrite to https")
	assert.Equal(t, csf.dnsCache["site2.org"]+":443", dummy.req.Host, "should use port 443")
	assert.Equal(t, fakeToken, dummy.req.Header.Get(common.CfgSvrAuthTokenHeader), "should attach token")
	assert.Equal(t, "", dummy.req.Header.Get(common.CfgSvrClientIPHeader), "should not attach client ip if remote address is invalid")

	req, _ = http.NewRequest("GET", "http://not-config-server.org/abc.gz", nil)
	req.RemoteAddr = dummyAddr
	chain.Apply(ctx, req, next)
	assert.Equal(t, "http", dummy.req.URL.Scheme, "should not rewrite to https for other sites")
	assert.Equal(t, "not-config-server.org", dummy.req.Host, "should not use port 443 for other sites")
	assert.Equal(t, "", dummy.req.Header.Get(common.CfgSvrAuthTokenHeader), "should not attach token for other sites")
	assert.Equal(t, "", dummy.req.Header.Get(common.CfgSvrClientIPHeader), "should not attach client ip for other sites")
}

func TestDialer(t *testing.T) {
	var address string
	dummyDial := func(net, addr string) (net.Conn, error) {
		address = addr
		return mockconn.SucceedingDialer([]byte{}).Dial(net, addr)
	}
	d := Dialer(dummyDial, &Options{"", []string{"site1", "site2"}})

	c, _ := d("tcp", "site1")
	_, ok := c.(*tls.Conn)
	assert.True(t, ok, "should override dialer if site is in list")
	c, _ = d("tcp", "other")
	_, ok = c.(*tls.Conn)
	assert.False(t, ok, "should not override dialer for other dialers")
}

func TestInitializeNoDomains(t *testing.T) {
	defer func() {
		r := recover()
		assert.NotNil(t, r, "should panic when no domains provided")
	}()
	New(&Options{AuthToken: "fake-token"})
}

func TestInitializeNoAuthToken(t *testing.T) {
	defer func() {
		r := recover()
		assert.NotNil(t, r, "should panic when no token provided")
	}()
	New(&Options{Domains: []string{"site1.com", "site2.org"}})
}
