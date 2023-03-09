package versioncheck

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/getlantern/proxy/v2"

	"github.com/getlantern/http-proxy-lantern/v2/common"
)

const (
	ip           = "8.8.8.8"
	redirectURL  = "https://versioncheck.com/badversion"
	redirectAddr = "versioncheck.com:443"
)

func TestParseVersionRange(t *testing.T) {
	_, e := New("> 3.1.x", "", nil, 1, nil)
	assert.NoError(t, e)
	_, e = New("< 3.x", "", nil, 1, nil)
	assert.NoError(t, e)
	_, e = New("= 3.1.1", "", nil, 1, nil)
	assert.NoError(t, e)
}

func TestRedirectRules(t *testing.T) {
	f, _ := New("< 3.1.x", redirectURL, nil, 1, nil)
	req, _ := http.NewRequest("POST", "http://anysite.com", nil)
	shouldRedirect := func() bool {
		should, _ := f.shouldRedirect(req)
		return should
	}
	assert.False(t, shouldRedirect(), "should not redirect POST requests")
	req, _ = http.NewRequest("CONNECT", "http://anysite.com", nil)
	assert.False(t, shouldRedirect(), "should not redirect CONNECT requests")
	req, _ = http.NewRequest("GET", "http://anysite.com", nil)
	assert.False(t, shouldRedirect(), "should not redirect non-HTML requests")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	assert.False(t, shouldRedirect(), "should only redirect requests from browser")
	req.Header.Set("User-Agent", "Mozilla/5.0 xxx")
	assert.False(t, shouldRedirect(), "should not redirect if no version header present")
	req.Header.Set("X-Lantern-Version", "development")
	assert.False(t, shouldRedirect(), "should not redirect if the version is not semantic")
	req.Header.Set("X-Lantern-Version", "3.1.0")
	assert.False(t, shouldRedirect(), "should not redirect if version equals to the min version")
	req.Header.Set("X-Lantern-Version", "3.1.1")
	assert.False(t, shouldRedirect(), "should not redirect if version is above the min version")
	req.Header.Set("X-Lantern-Version", "3.11.0")
	assert.False(t, shouldRedirect(), "should not redirect if version is above the min version")
	req.Header.Set("X-Lantern-Version", "3.0.1")
	assert.True(t, shouldRedirect(), "should redirect if version is below the min version")
	req.Header.Set("X-Lantern-App", "not-lantern")
	assert.True(t, shouldRedirect(), "should redirect even if the request is not from Lantern")
	req.Header.Set("X-Lantern-App", "Lantern")
	assert.True(t, shouldRedirect(), "should check app name case-insensitively")
}

func TestPercentage(t *testing.T) {
	testPercentage(t, 1, true)
	testPercentage(t, 0.1, false)
	testPercentage(t, float64(1)/1000, false)
}

func testPercentage(t *testing.T, percentage float64, exact bool) {
	f, _ := New("< 3.1.1", redirectURL, nil, percentage, nil)
	req, _ := http.NewRequest("GET", "http://anysite.com", nil)
	req.Header.Set("Accept", "text/html")
	req.Header.Set("User-Agent", "Mozilla/5.0 xxx")
	req.Header.Set("X-Lantern-Version", "3.0.1")
	hit := 0
	expected := int(percentage * oneMillion)
	for i := 0; i < oneMillion; i++ {
		if should, _ := f.shouldRedirect(req); should {
			hit++
		}
	}
	if exact {
		assert.Equal(t, expected, hit)
	} else {
		assert.InDelta(t, expected, hit, float64(expected)/10)
	}
}

func TestRedirectGet(t *testing.T) {
	originURL, proxyAddr, close := setupServers(t)
	defer close()
	req, _ := http.NewRequest("GET", originURL, nil)
	req.Header.Set("Accept", "text/html whatever")
	req.Header.Set("User-Agent", "Mozilla/5.0 xxx")
	req.Header.Set("X-Lantern-Version", "3.1.0")
	r, err := requestViaProxy(t, req, proxyAddr, "", false)
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, http.StatusFound, r.StatusCode,
		"should redirect with matched conditions")
	assert.Equal(t, r.Header.Get("Location"), redirectURL)
	b, _ := ioutil.ReadAll(r.Body)
	assert.Equal(t, "", string(b))
}

func TestRedirectConnect(t *testing.T) {
	originURL, proxyAddr, close := setupServers(t)
	defer close()
	proxiedReq, _ := http.NewRequest("GET", originURL, nil)
	r, err := requestViaProxy(t, proxiedReq, proxyAddr, "", true)
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, http.StatusOK, r.StatusCode,
		"should not redirect when no version header is present")
	b, _ := ioutil.ReadAll(r.Body)
	assert.Equal(t, "good", string(b))

	r, err = requestViaProxy(t, proxiedReq, proxyAddr, "3.1.0", true)
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, http.StatusFound, r.StatusCode,
		"should redirect when version is lower than expected")
	assert.Equal(t, r.Header.Get("Location"), redirectURL)
	b, _ = ioutil.ReadAll(r.Body)
	assert.Equal(t, "", string(b))

	r, err = requestViaProxy(t, proxiedReq, proxyAddr, "3.1.1", true)
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, http.StatusOK, r.StatusCode,
		"should not redirect when version is equal to or higher than expected")
	assert.Equal(t, r.Header.Get("Location"), "")
	b, _ = ioutil.ReadAll(r.Body)
	assert.Equal(t, "good", string(b))
}

func setupServers(t *testing.T) (string, string, func()) {
	originServer := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.Write([]byte("good"))
	}))

	originURL, _ := url.Parse(originServer.URL)
	_, originPort, _ := net.SplitHostPort(originURL.Host)
	l, err := net.Listen("tcp", "localhost:0")
	if !assert.NoError(t, err) {
		return "", "", originServer.Close
	}

	f, _ := New("< 3.1.1", redirectURL, []string{originPort}, 1, nil)
	p, _ := proxy.New(&proxy.Opts{Filter: f})
	go p.Serve(l)
	return originServer.URL, l.Addr().String(), func() {
		originServer.Close()
		l.Close()
	}
}

func requestViaProxy(t *testing.T, proxiedReq *http.Request, proxyAddr, version string, withCONNECT bool) (*http.Response, error) {
	proxyConn, err := net.Dial("tcp", proxyAddr)
	if !assert.NoError(t, err) {
		return nil, err
	}
	defer proxyConn.Close()
	var buf bytes.Buffer
	bufReader := bufio.NewReader(io.TeeReader(proxyConn, &buf))
	if withCONNECT {
		req, err := http.NewRequest("CONNECT", "http://"+proxiedReq.Host, nil)
		if err != nil {
			return nil, fmt.Errorf("Unable to construct CONNECT request: %v", err)
		}
		if version != "" {
			req.Header.Add(common.VersionHeader, version)
		}
		err = req.Write(proxyConn)
		if err != nil {
			return nil, fmt.Errorf("Unable to issue CONNECT request: %v", err)
		}
		resp, err := http.ReadResponse(bufReader, req)
		if err != nil {
			return nil, fmt.Errorf("Unable to read CONNECT response: %v", err)
		}
		assert.Equal(t, http.StatusOK, resp.StatusCode, "proxy should respond 200 OK")
	}
	err = proxiedReq.Write(proxyConn)
	if err != nil {
		return nil, fmt.Errorf("Unable to issue proxied request: %v", err)
	}
	resp, err := http.ReadResponse(bufReader, proxiedReq)
	// Ignore EOF, as that's an OK error
	if err == io.EOF {
		err = nil
	}
	return resp, err
}
