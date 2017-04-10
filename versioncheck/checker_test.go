package versioncheck

import (
	"bufio"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/getlantern/http-proxy/filters"
	"github.com/getlantern/http-proxy/httpconnect"
	"github.com/stretchr/testify/assert"

	"github.com/getlantern/http-proxy-lantern/common"
)

const (
	ip = "8.8.8.8"
)

func TestRewrite(t *testing.T) {
	rewriteURL := "https://versioncheck.com/badversion"
	rewriteAddr := "versioncheck.com:443"
	f := New("3.1.1", rewriteURL, nil, 1)
	req, _ := http.NewRequest("POST", "http://anysite.com", nil)
	assert.False(t, f.shouldRewrite(req), "should only rewrite GET requests")
	req, _ = http.NewRequest("GET", "http://anysite.com", nil)
	assert.False(t, f.shouldRewrite(req), "should only rewrite HTML requests")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	assert.False(t, f.shouldRewrite(req), "should only rewrite requests from browser")
	req.Header.Set("User-Agent", "Mozilla/5.0 xxx")
	assert.True(t, f.shouldRewrite(req), "should rewrite if no version header present")
	req.Header.Set("X-Lantern-Version", "development")
	assert.True(t, f.shouldRewrite(req), "should rewrite if the version is not semantic")
	req.Header.Set("X-Lantern-Version", "3.1.1")
	assert.False(t, f.shouldRewrite(req), "should not rewrite if version equals to the min version")
	req.Header.Set("X-Lantern-Version", "3.1.2")
	assert.False(t, f.shouldRewrite(req), "should not rewrite if version is above the min version")
	req.Header.Set("X-Lantern-Version", "3.11.0")
	assert.False(t, f.shouldRewrite(req), "should not rewrite if version is above the min version")
	req.Header.Set("X-Lantern-Version", "3.1.0")
	assert.True(t, f.shouldRewrite(req), "should rewrite if version is below the min version")

	f.RewriteIfNecessary(req)
	assert.Equal(t, rewriteURL, req.URL.String())
	assert.Equal(t, rewriteAddr, req.Host)
}

func TestPercentage(t *testing.T) {
	testPercentage(t, 1, true)
	testPercentage(t, 0.1, false)
	testPercentage(t, float64(1)/1000, false)
}

func testPercentage(t *testing.T, percentage float64, exact bool) {
	f := New("3.1.1", "http://versioncheck.com/badversion", nil, percentage)
	req, _ := http.NewRequest("GET", "http://anysite.com", nil)
	req.Header.Set("Accept", "text/html")
	req.Header.Set("User-Agent", "Mozilla/5.0 xxx")
	hit := 0
	expected := int(percentage * oneMillion)
	for i := 0; i < oneMillion; i++ {
		if f.shouldRewrite(req) {
			hit++
		}
	}
	if exact {
		assert.Equal(t, expected, hit)
	} else {
		assert.InDelta(t, expected, hit, float64(expected)/10)
	}
}

func TestRedirectConnect(t *testing.T) {
	originServer := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.Write([]byte("good"))
	}))
	defer originServer.Close()

	originURL, _ := url.Parse(originServer.URL)
	_, originPort, _ := net.SplitHostPort(originURL.Host)
	rewriteURL := "https://versioncheck.com/badversion"
	proxy := httptest.NewServer(filters.Join(
		New("3.1.1", rewriteURL, []string{originPort}, 1),
		httpconnect.New(&httpconnect.Options{
			Dialer:      net.Dial,
			IdleTimeout: 1 * time.Second,
		}),
	))
	defer proxy.Close()

	proxiedReq, _ := http.NewRequest("GET", originServer.URL, nil)
	r := requestViaProxy(t, proxiedReq, proxy, "")
	if assert.NotNil(t, r) {
		assert.Equal(t, http.StatusFound, r.StatusCode,
			"should redirect when no version header is present")
		assert.Equal(t, r.Header.Get("Location"), rewriteURL)
		b, _ := ioutil.ReadAll(r.Body)
		assert.Equal(t, "", string(b))
	}

	r = requestViaProxy(t, proxiedReq, proxy, "3.1.0")
	if assert.NotNil(t, r) {
		assert.Equal(t, http.StatusFound, r.StatusCode,
			"should redirect when version is lower than expected")
		assert.Equal(t, r.Header.Get("Location"), rewriteURL)
		b, _ := ioutil.ReadAll(r.Body)
		assert.Equal(t, "", string(b))
	}

	r = requestViaProxy(t, proxiedReq, proxy, "3.1.1")
	if assert.NotNil(t, r) {
		assert.Equal(t, http.StatusOK, r.StatusCode,
			"should not redirect when version is equal to or higher than expected")
		assert.Equal(t, r.Header.Get("Location"), "")
		b, _ := ioutil.ReadAll(r.Body)
		assert.Equal(t, "good", string(b))
	}
}

func requestViaProxy(t *testing.T, proxiedReq *http.Request, proxy *httptest.Server, version string) *http.Response {
	proxyConn, _ := net.Dial("tcp", proxy.Listener.Addr().String())
	defer proxyConn.Close()
	req, err := http.NewRequest("CONNECT", "http://"+proxiedReq.Host, nil)
	assert.NoError(t, err)
	if version != "" {
		req.Header.Add(common.VersionHeader, version)
	}
	req.Write(proxyConn)
	resp, err := http.ReadResponse(bufio.NewReader(proxyConn), nil)
	if !assert.NoError(t, err, "should have received proxy's response") {
		return nil
	}
	assert.Equal(t, http.StatusOK, resp.StatusCode, "proxy should respond 200 OK")
	proxiedReq.Write(proxyConn)
	r, e := http.ReadResponse(bufio.NewReader(proxyConn), nil)
	if assert.NoError(t, e, "should have received proxied response") {
		return r
	}
	return nil
}
