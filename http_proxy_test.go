package proxy

import (
	"bufio"
	"crypto/tls"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/getlantern/keyman"
	"github.com/getlantern/measured"

	"github.com/getlantern/http-proxy-lantern/v2/common"
	"github.com/getlantern/http-proxy-lantern/v2/listeners"
	"github.com/getlantern/http-proxy-lantern/v2/server"

	"github.com/getlantern/http-proxy-lantern/v2/instrument"
	"github.com/getlantern/http-proxy-lantern/v2/tokenfilter"
)

const (
	deviceId       = "1234-1234-1234-1234-1234-1234"
	validToken     = "6o0dToK3n"
	tunneledReq    = "GET / HTTP/1.1\r\nHost:localhost\r\n\r\n"
	targetResponse = "Fight for a Free Internet!"
)

var (
	mr = &mockReporter{traffic: make(map[string]*measured.Stats)}

	httpProxyAddr    string
	tlsProxyAddr     string
	httpTargetServer *targetHandler
	httpTargetURL    string
	tlsTargetServer  *targetHandler
	tlsTargetURL     string

	serverCertificate *keyman.Certificate
	// TODO: this should be imported from tlsdefaults package, but is not being
	// exported there.
	preferredCipherSuites = []uint16{
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
		tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
		tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
		tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
	}
)

func TestMain(m *testing.M) {
	flag.Parse()
	var err error

	// Set up mock target servers
	httpTargetURL, httpTargetServer = newTargetHandler(targetResponse, false)
	defer httpTargetServer.Close()
	tlsTargetURL, tlsTargetServer = newTargetHandler(targetResponse, true)
	defer tlsTargetServer.Close()

	// Set up HTTP chained server
	httpProxyAddr, err = setupNewHTTPServer(0, 30*time.Second, false)
	if err != nil {
		log.Fatalf("Error starting proxy server: %v", err)
	}
	log.Debugf("Started HTTP proxy server at %s", httpProxyAddr)

	// Set up HTTPS chained server
	tlsProxyAddr, err = setupNewHTTPServer(0, 30*time.Second, true)
	if err != nil {
		log.Fatalf("Error starting proxy server: %v", err)
	}
	log.Debugf("Started HTTPS proxy server at %s", tlsProxyAddr)

	os.Exit(m.Run())
}

func TestProxyNameAndDC(t *testing.T) {
	runTest := func(name string, expectedName, expectedDatacenter string) {
		resultName, resultDC := proxyNameAndDC(name)
		assert.Equal(t, expectedName, resultName)
		assert.Equal(t, expectedDatacenter, resultDC)
	}

	runTest("fp-https-donyc3-20180101-006-kcp", "fp-https-donyc3-20180101-006", "donyc3")
	runTest("fp-donyc3-20180101-006-kcp", "fp-donyc3-20180101-006", "donyc3")
	runTest("fp-donyc3-20180101-006", "fp-donyc3-20180101-006", "donyc3")
	runTest("fp-obfs4-donyc3-20160715-005", "fp-obfs4-donyc3-20160715-005", "donyc3")
	runTest("fp-14325-adsfds-006", "fp-14325-adsfds-006", "")
	runTest("cloudcompile", "cloudcompile", "")
}

// Keep this one first to avoid measuring previous connections
func TestReportStats(t *testing.T) {
	connectReq := "CONNECT %s HTTP/1.1\r\nHost: %s\r\nX-Lantern-Device-Id: %s\r\n\r\n"
	connectResp := "HTTP/1.1 400 Bad Request\r\n"
	testFn := func(conn net.Conn, targetURL *url.URL) {
		var err error
		req := fmt.Sprintf(connectReq, targetURL.Host, targetURL.Host, deviceId)
		t.Log("\n" + req)
		_, err = conn.Write([]byte(req))
		if !assert.NoError(t, err, "should write CONNECT request") {
			t.FailNow()
		}

		var buf [400]byte
		_, err = conn.Read(buf[:])
		if !assert.Contains(t, string(buf[:]), connectResp,
			"should mimic Apache because no token was provided") {
			t.FailNow()
		}
	}

	testRoundTrip(t, httpProxyAddr, false, httpTargetServer, testFn)
	testRoundTrip(t, tlsProxyAddr, true, httpTargetServer, testFn)
	time.Sleep(200 * time.Millisecond)
	mr.tmtx.Lock()
	defer mr.tmtx.Unlock()
	if assert.True(t, len(mr.traffic) > 0) {
		stats := mr.traffic[""]
		if assert.NotNil(t, stats) {
			log.Debug(stats)
		}
	}
}

func TestMaxConnections(t *testing.T) {
	connectReq := "CONNECT %s HTTP/1.1\r\nHost: %s\r\nX-Lantern-Auth-Token: %s\r\nX-Lantern-Device-Id: %s\r\n\r\n"

	limitedServerAddr, err := setupNewHTTPServer(5, 30*time.Second, false)
	if err != nil {
		assert.Fail(t, "Error starting proxy server")
	}

	//limitedServer.httpServer.SetKeepAlivesEnabled(false)
	okFn := func(conn net.Conn, targetURL *url.URL) {
		req := fmt.Sprintf(connectReq, targetURL.Host, targetURL.Host, validToken, deviceId)
		conn.Write([]byte(req))

		var buf [400]byte
		_, err := conn.Read(buf[:])
		assert.NoError(t, err)

		time.Sleep(time.Millisecond * 100)
	}

	waitFn := func(conn net.Conn, targetURL *url.URL) {
		conn.SetReadDeadline(time.Now().Add(50 * time.Millisecond))

		req := fmt.Sprintf(connectReq, targetURL.Host, targetURL.Host, validToken, deviceId)
		conn.Write([]byte(req))

		var buf [400]byte
		_, err := conn.Read(buf[:])
		if assert.Error(t, err) {
			e, ok := err.(*net.OpError)

			assert.True(t, ok && e.Timeout(), "should be a time out error")
		}
	}

	for i := 0; i < 5; i++ {
		go testRoundTrip(t, limitedServerAddr, false, httpTargetServer, okFn)
	}

	time.Sleep(time.Millisecond * 50)

	for i := 0; i < 5; i++ {
		go testRoundTrip(t, limitedServerAddr, false, httpTargetServer, waitFn)
	}

	time.Sleep(time.Millisecond * 100)

	for i := 0; i < 5; i++ {
		go testRoundTrip(t, limitedServerAddr, false, httpTargetServer, okFn)
	}
}

func bufEmpty(buf [400]byte) bool {
	for _, c := range buf {
		if c != 0 {
			return false
		}
	}
	return true
}

func TestIdleClientConnections(t *testing.T) {
	limitedServerAddr, err := setupNewHTTPServer(0, 100*time.Millisecond, false)
	if err != nil {
		assert.Fail(t, "Error starting proxy server")
	}

	okFn := func(conn net.Conn, targetURL *url.URL) {
		time.Sleep(time.Millisecond * 90)
		conn.Write([]byte("GET / HTTP/1.1\r\n\r\n"))

		var buf [400]byte
		_, err := conn.Read(buf[:])

		assert.NoError(t, err)
	}

	idleFn := func(conn net.Conn, targetURL *url.URL) {
		time.Sleep(time.Millisecond * 110)
		conn.Write([]byte("GET / HTTP/1.1\r\n\r\n"))

		var buf [400]byte
		conn.Read(buf[:])

		assert.True(t, bufEmpty(buf), "Should fail")
	}

	go testRoundTrip(t, limitedServerAddr, false, httpTargetServer, okFn)
	testRoundTrip(t, limitedServerAddr, false, httpTargetServer, idleFn)
}

// TODO: Since both client and target server idle timeouts are identical,
// we are just testing the combined behavior.  We probably can do that by
// creating a custom server that only sets one timeout at a time
func TestIdleTargetConnections(t *testing.T) {
	impatientTimeout := 30 * time.Millisecond
	normalServerAddr, err := setupNewHTTPServer(0, 30*time.Second, false)
	if err != nil {
		assert.Fail(t, "Error starting proxy server: %s", err)
	}

	impatientServerAddr, err := setupNewHTTPServer(0, impatientTimeout, false)
	if err != nil {
		assert.Fail(t, "Error starting proxy server: %s", err)
	}

	okForwardFn := func(conn net.Conn, targetURL *url.URL) {
		conn.Write([]byte("GET / HTTP/1.1\r\n\r\n"))
		var buf [400]byte
		_, err := conn.Read(buf[:])

		assert.NoError(t, err)
	}

	okConnectFn := func(conn net.Conn, targetURL *url.URL) {
		conn.Write([]byte("CONNECT www.google.com HTTP/1.1\r\nHost: www.google.com\r\n\r\n"))
		var buf [400]byte
		_, err := conn.Read(buf[:])

		assert.NoError(t, err)
	}

	failForwardFn := func(conn net.Conn, targetURL *url.URL) {
		reqStr := "GET / HTTP/1.1\r\nHost: %s\r\nX-Lantern-Auth-Token: %s\r\nX-Lantern-Device-Id: %s\r\n\r\n"
		req := fmt.Sprintf(reqStr, targetURL.Host, validToken, deviceId)
		t.Log("\n" + req)

		time.Sleep(impatientTimeout * 2)
		conn.Write([]byte("GET / HTTP/1.1\r\n\r\n"))
		var buf [400]byte
		conn.Read(buf[:])

		assert.True(t, bufEmpty(buf), "Should fail")
	}

	failConnectFn := func(conn net.Conn, targetURL *url.URL) {
		reqStr := "CONNECT www.google.com HTTP/1.1\r\nHost: www.google.com\r\nX-Lantern-Auth-Token: %s\r\nX-Lantern-Device-Id: %s\r\n\r\n"
		req := fmt.Sprintf(reqStr, validToken, deviceId)
		conn.Write([]byte(req))
		var buf [400]byte
		conn.Read(buf[:])

		time.Sleep(impatientTimeout * 10)
		conn.Write([]byte("GET / HTTP/1.1\r\n\r\n"))
		_, err := http.ReadResponse(bufio.NewReader(conn), nil)

		assert.Error(t, err)
	}

	testRoundTrip(t, normalServerAddr, false, httpTargetServer, okForwardFn)
	testRoundTrip(t, normalServerAddr, false, httpTargetServer, okConnectFn)
	testRoundTrip(t, impatientServerAddr, false, httpTargetServer, failForwardFn)
	testRoundTrip(t, impatientServerAddr, false, httpTargetServer, failConnectFn)
}

// No X-Lantern-Auth-Token -> 400
func TestConnectNoToken(t *testing.T) {
	connectReq := "CONNECT %s HTTP/1.1\r\nHost: %s\r\nX-Lantern-Device-Id: %s\r\n\r\n"
	connectResp := "HTTP/1.1 400 Bad Request\r\n"

	testFn := func(conn net.Conn, targetURL *url.URL) {
		var err error
		req := fmt.Sprintf(connectReq, targetURL.Host, targetURL.Host, deviceId)
		t.Log("\n" + req)
		_, err = conn.Write([]byte(req))
		if !assert.NoError(t, err, "should write CONNECT request") {
			t.FailNow()
		}

		var buf [400]byte
		_, err = conn.Read(buf[:])
		if !assert.Contains(t, string(buf[:]), connectResp,
			"should mimic Apache because no token was provided") {
			t.FailNow()
		}
	}

	testRoundTrip(t, httpProxyAddr, false, httpTargetServer, testFn)
	testRoundTrip(t, tlsProxyAddr, true, httpTargetServer, testFn)

	testRoundTrip(t, httpProxyAddr, false, tlsTargetServer, testFn)
	testRoundTrip(t, tlsProxyAddr, true, tlsTargetServer, testFn)
}

// Bad X-Lantern-Auth-Token -> 400
func TestConnectBadToken(t *testing.T) {
	connectReq := "CONNECT %s HTTP/1.1\r\nHost: %s\r\nX-Lantern-Auth-Token: %s\r\nX-Lantern-Device-Id: %s\r\n\r\n"
	connectResp := "HTTP/1.1 400 Bad Request\r\n"

	testFn := func(conn net.Conn, targetURL *url.URL) {
		var err error
		req := fmt.Sprintf(connectReq, targetURL.Host, targetURL.Host, "B4dT0k3n", deviceId)
		t.Log("\n" + req)
		_, err = conn.Write([]byte(req))
		if !assert.NoError(t, err, "should write CONNECT request") {
			t.FailNow()
		}

		var buf [400]byte
		_, err = conn.Read(buf[:])
		if !assert.Contains(t, string(buf[:]), connectResp,
			"should mimic Apache because no token was provided") {
			t.FailNow()
		}
	}

	testRoundTrip(t, httpProxyAddr, false, httpTargetServer, testFn)
	testRoundTrip(t, tlsProxyAddr, true, httpTargetServer, testFn)

	testRoundTrip(t, httpProxyAddr, false, tlsTargetServer, testFn)
	testRoundTrip(t, tlsProxyAddr, true, tlsTargetServer, testFn)
}

// No X-Lantern-Device-Id -> 400
func TestConnectNoDevice(t *testing.T) {
	// TODO: Deactivated because this filter is deactivated
	t.SkipNow()

	connectReq := "CONNECT %s HTTP/1.1\r\nHost: %s\r\nX-Lantern-Auth-Token: %s\r\n\r\n"
	connectResp := "HTTP/1.1 400 Bad Request\r\n"

	testFn := func(conn net.Conn, targetURL *url.URL) {
		var err error
		req := fmt.Sprintf(connectReq, targetURL.Host, targetURL.Host, validToken)
		t.Log("\n" + req)
		_, err = conn.Write([]byte(req))
		if !assert.NoError(t, err, "should write CONNECT request") {
			t.FailNow()
		}

		var buf [400]byte
		_, err = conn.Read(buf[:])
		if !assert.Contains(t, string(buf[:]), connectResp,
			"should mimic Apache because no token was provided") {
			t.FailNow()
		}
	}

	testRoundTrip(t, httpProxyAddr, false, httpTargetServer, testFn)
	testRoundTrip(t, tlsProxyAddr, true, httpTargetServer, testFn)

	testRoundTrip(t, httpProxyAddr, false, tlsTargetServer, testFn)
	testRoundTrip(t, tlsProxyAddr, true, tlsTargetServer, testFn)
}

// X-Lantern-Auth-Token + X-Lantern-Device-Id -> 200 OK <- Tunneled request -> 200 OK
func TestConnectOK(t *testing.T) {
	connectReq := "CONNECT %s HTTP/1.1\r\nHost: %s\r\nX-Lantern-Auth-Token: %s\r\nX-Lantern-Device-Id: %s\r\n\r\n"

	testHTTP := func(conn net.Conn, targetURL *url.URL) {
		req := fmt.Sprintf(connectReq, targetURL.Host, targetURL.Host, validToken, deviceId)
		t.Log("\n" + req)
		_, err := conn.Write([]byte(req))
		if !assert.NoError(t, err, "should write CONNECT request") {
			t.FailNow()
		}

		resp, _ := http.ReadResponse(bufio.NewReader(conn), nil)
		buf, _ := ioutil.ReadAll(resp.Body)
		if !assert.Equal(t, 200, resp.StatusCode) {
			t.FailNow()
		}

		_, err = conn.Write([]byte(tunneledReq))
		if !assert.NoError(t, err, "should write tunneled data") {
			t.FailNow()
		}

		resp, _ = http.ReadResponse(bufio.NewReader(conn), nil)
		buf, _ = ioutil.ReadAll(resp.Body)
		assert.Contains(t, string(buf[:]), targetResponse, "should read tunneled response")
	}

	testTLS := func(conn net.Conn, targetURL *url.URL) {
		req := fmt.Sprintf(connectReq, targetURL.Host, targetURL.Host, validToken, deviceId)
		t.Log("\n" + req)
		_, err := conn.Write([]byte(req))
		if !assert.NoError(t, err, "should write CONNECT request") {
			t.FailNow()
		}

		resp, _ := http.ReadResponse(bufio.NewReader(conn), nil)
		buf, _ := ioutil.ReadAll(resp.Body)
		if !assert.Equal(t, 200, resp.StatusCode) {
			t.FailNow()
		}

		// HTTPS-Tunneled HTTPS
		tunnConn := tls.Client(conn, &tls.Config{
			InsecureSkipVerify: true,
		})
		tunnConn.Handshake()

		_, err = tunnConn.Write([]byte(tunneledReq))
		if !assert.NoError(t, err, "should write tunneled data") {
			t.FailNow()
		}

		resp, _ = http.ReadResponse(bufio.NewReader(tunnConn), nil)
		buf, _ = ioutil.ReadAll(resp.Body)
		assert.Contains(t, string(buf[:]), targetResponse, "should read tunneled response")
	}

	testRoundTrip(t, httpProxyAddr, false, httpTargetServer, testHTTP)
	testRoundTrip(t, tlsProxyAddr, true, httpTargetServer, testHTTP)

	testRoundTrip(t, httpProxyAddr, false, tlsTargetServer, testTLS)
	testRoundTrip(t, tlsProxyAddr, true, tlsTargetServer, testTLS)
}

// No X-Lantern-Auth-Token -> 404
func TestDirectNoToken(t *testing.T) {
	connectReq := "GET /%s HTTP/1.1\r\nHost: %s\r\nX-Lantern-Device-Id: %s\r\n\r\n"
	connectResp := "HTTP/1.1 404 Not Found\r\n"

	testFn := func(conn net.Conn, targetURL *url.URL) {
		var err error
		req := fmt.Sprintf(connectReq, targetURL.Host, targetURL.Host, deviceId)
		t.Log("\n" + req)
		_, err = conn.Write([]byte(req))
		if !assert.NoError(t, err, "should write CONNECT request") {
			t.FailNow()
		}

		var buf [400]byte
		_, err = conn.Read(buf[:])
		if !assert.Contains(t, string(buf[:]), connectResp,
			"should get 404 Not Found because no token was provided") {
			t.FailNow()
		}
	}

	testRoundTrip(t, httpProxyAddr, false, httpTargetServer, testFn)
	testRoundTrip(t, tlsProxyAddr, true, httpTargetServer, testFn)

	testRoundTrip(t, httpProxyAddr, false, tlsTargetServer, testFn)
	testRoundTrip(t, tlsProxyAddr, true, tlsTargetServer, testFn)
}

// Bad X-Lantern-Auth-Token -> 404
func TestDirectBadToken(t *testing.T) {
	connectReq := "GET /%s HTTP/1.1\r\nHost: %s\r\nX-Lantern-Auth-Token: %s\r\nX-Lantern-Device-Id: %s\r\n\r\n"
	connectResp := "HTTP/1.1 404 Not Found\r\n"

	testFn := func(conn net.Conn, targetURL *url.URL) {
		var err error
		req := fmt.Sprintf(connectReq, targetURL.Host, targetURL.Host, "B4dT0k3n", deviceId)
		t.Log("\n" + req)
		_, err = conn.Write([]byte(req))
		if !assert.NoError(t, err, "should write CONNECT request") {
			t.FailNow()
		}

		var buf [400]byte
		_, err = conn.Read(buf[:])
		if !assert.Contains(t, string(buf[:]), connectResp,
			"should get 404 Not Found because no token was provided") {
			t.FailNow()
		}
	}

	testRoundTrip(t, httpProxyAddr, false, httpTargetServer, testFn)
	testRoundTrip(t, tlsProxyAddr, true, httpTargetServer, testFn)

	testRoundTrip(t, httpProxyAddr, false, tlsTargetServer, testFn)
	testRoundTrip(t, tlsProxyAddr, true, tlsTargetServer, testFn)
}

// No X-Lantern-Device-Id -> 404
func TestDirectNoDevice(t *testing.T) {
	// TODO: Deactivated because this filter is deactivated
	t.SkipNow()

	connectReq := "GET /%s HTTP/1.1\r\nHost: %s\r\nX-Lantern-Auth-Token: %s\r\n\r\n"
	connectResp := "HTTP/1.1 404 Not Found\r\n"

	testFn := func(conn net.Conn, targetURL *url.URL) {
		var err error
		req := fmt.Sprintf(connectReq, targetURL.Host, targetURL.Host, validToken)
		t.Log("\n" + req)
		_, err = conn.Write([]byte(req))
		if !assert.NoError(t, err, "should write CONNECT request") {
			t.FailNow()
		}

		var buf [400]byte
		conn.Read(buf[:])
		if !assert.Contains(t, string(buf[:]), connectResp,
			"should get 404 Not Found because no token was provided") {
			t.FailNow()
		}
	}

	testRoundTrip(t, httpProxyAddr, false, httpTargetServer, testFn)
	testRoundTrip(t, tlsProxyAddr, true, httpTargetServer, testFn)

	testRoundTrip(t, httpProxyAddr, false, tlsTargetServer, testFn)
	testRoundTrip(t, tlsProxyAddr, true, tlsTargetServer, testFn)
}

// X-Lantern-Auth-Token + X-Lantern-Device-Id -> Forward
func TestDirectOK(t *testing.T) {
	buildRequest := func(url *url.URL, token string, deviceID string) *http.Request {
		req, _ := http.NewRequest(http.MethodGet, url.String(), nil)
		req.Header.Add("X-Lantern-Auth-Token", "extraauthtoken")
		req.Header.Add("X-Lantern-Auth-Token", token)
		req.Header.Set("X-Lantern-Device-Id", deviceID)
		return req
	}

	testOk := func(conn net.Conn, targetURL *url.URL) {
		req := buildRequest(targetURL, validToken, deviceId)
		err := req.Write(conn)
		if !assert.NoError(t, err, "should write GET request") {
			t.FailNow()
		}

		resp, err := http.ReadResponse(bufio.NewReader(conn), req)
		if !assert.NoError(t, err) {
			t.FailNow()
		}
		body, err := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		if !assert.NoError(t, err) {
			t.FailNow()
		}
		assert.Contains(t, string(body), targetResponse, "should read tunneled response")
	}

	testFail := func(conn net.Conn, targetURL *url.URL) {
		req := buildRequest(targetURL, validToken, deviceId)
		err := req.Write(conn)
		if !assert.NoError(t, err, "should write GET request") {
			t.FailNow()
		}

		resp, err := http.ReadResponse(bufio.NewReader(conn), req)
		if !assert.NoError(t, err) {
			t.FailNow()
		}
		if !assert.Equal(t, http.StatusBadRequest, resp.StatusCode) {
			resp.Write(os.Stdout)
		}
	}

	testRoundTrip(t, httpProxyAddr, false, httpTargetServer, testOk)
	testRoundTrip(t, tlsProxyAddr, true, httpTargetServer, testOk)

	// HTTPS can't be tunneled using Direct Proxying, as redirections
	// require a TLS handshake between the proxy and the target
	testRoundTrip(t, httpProxyAddr, false, tlsTargetServer, testFail)
	testRoundTrip(t, tlsProxyAddr, true, tlsTargetServer, testFail)
}

func TestInvalidRequest(t *testing.T) {
	connectResp := "HTTP/1.1 400 Bad Request\r\n"
	testFn := func(conn net.Conn, targetURL *url.URL) {
		_, err := conn.Write([]byte("GET HTTP/1.1\r\n\r\n"))
		if !assert.NoError(t, err, "should write GET request") {
			t.FailNow()
		}

		buf := [400]byte{}
		conn.Read(buf[:])
		assert.Contains(t, string(buf[:]), connectResp, "should 400")

	}
	for i := 0; i < 10; i++ {
		testRoundTrip(t, httpProxyAddr, false, tlsTargetServer, testFn)
		testRoundTrip(t, tlsProxyAddr, true, tlsTargetServer, testFn)
	}

}

func TestPortsFromCSV(t *testing.T) {
	ports, err := portsFromCSV("1, 2, 3,4,5")
	if assert.NoError(t, err, "should have no error parsing CSV") {
		assert.Equal(t, []int{1, 2, 3, 4, 5}, ports, "should get correct ports")
	}
}

//
// Auxiliary functions
//

func testRoundTrip(t *testing.T, addr string, isTls bool, target *targetHandler, checkerFn func(conn net.Conn, targetURL *url.URL)) {
	var conn net.Conn
	var err error

	if !isTls {
		conn, err = net.Dial("tcp", addr)
		log.Debugf("%s -> %s (via HTTP) -> %s", conn.LocalAddr().String(), addr, target.server.URL)
		if !assert.NoError(t, err, "should dial proxy server") {
			t.FailNow()
		}
	} else {
		var tlsConn *tls.Conn
		x509cert := serverCertificate.X509()
		tlsConn, err = tls.Dial("tcp", addr, &tls.Config{
			CipherSuites:       preferredCipherSuites,
			InsecureSkipVerify: true,
		})
		log.Debugf("%s -> %s (via HTTPS) -> %s", tlsConn.LocalAddr().String(), addr, target.server.URL)
		if !assert.NoError(t, err, "should dial proxy server") {
			t.FailNow()
		}
		conn = tlsConn
		if !tlsConn.ConnectionState().PeerCertificates[0].Equal(x509cert) {
			if err := tlsConn.Close(); err != nil {
				log.Errorf("Error closing chained server connection: %s", err)
			}
			t.Fatal("Server's certificate didn't match expected")
		}
	}
	defer func() {
		assert.NoError(t, conn.Close(), "should close connection")
	}()

	url, _ := url.Parse(target.server.URL)
	checkerFn(conn, url)
}

func basicServer(maxConns uint64, idleTimeout time.Duration) *server.Server {
	// Create server
	srv := server.New(&server.Opts{
		IdleTimeout: idleTimeout,
		Filter:      tokenfilter.New(validToken, instrument.NoInstrument{}),
	})

	// Add net.Listener wrappers for inbound connections
	srv.AddListenerWrappers(
		// Limit max number of simultaneous connections
		// We test this even if not used in production, it serves as a watchdog
		// on the status of stacked connection wrappers
		func(ls net.Listener) net.Listener {
			return listeners.NewLimitedListener(ls, maxConns)
		},
		// Close connections after idleTimeout seconds of no activity
		func(ls net.Listener) net.Listener {
			return listeners.NewIdleConnListener(ls, idleTimeout)
		},
		// Measure connections
		func(ls net.Listener) net.Listener {
			return listeners.NewMeasuredListener(ls, 100*time.Millisecond, mr.Report)
		},
	)

	return srv
}

func setupNewHTTPServer(maxConns uint64, idleTimeout time.Duration, https bool) (addr string, err error) {
	var (
		s     = basicServer(maxConns, idleTimeout)
		ready = make(chan string, 1)
		errC  = make(chan error, 1)
		wait  = func(addr string) { ready <- addr }
	)
	go func() {
		if https {
			errC <- s.ListenAndServeHTTPS("localhost:0", "key.pem", "cert.pem", wait)
		} else {
			errC <- s.ListenAndServeHTTP("localhost:0", wait)
		}
	}()
	select {
	case addr = <-ready:
		if https {
			serverCertificate, err = keyman.LoadCertificateFromFile("cert.pem")
		}
	case err = <-errC:
	}
	return
}

//
// Mock target server
// Emulating locally a target site for testing tunnels
//

type targetHandler struct {
	writer func(w http.ResponseWriter)
	server *httptest.Server
}

func (m *targetHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.writer(w)
}

func (m *targetHandler) Raw(msg string) {
	m.writer = func(w http.ResponseWriter) {
		conn, _, _ := w.(http.Hijacker).Hijack()
		if _, err := conn.Write([]byte(msg)); err != nil {
			log.Errorf("Unable to write to connection: %v", err)
		}
		if err := conn.Close(); err != nil {
			log.Errorf("Unable to close connection: %v", err)
		}
	}
}

func (m *targetHandler) Msg(msg string) {
	m.writer = func(w http.ResponseWriter) {
		w.Header()["Content-Length"] = []string{strconv.Itoa(len(msg))}
		_, _ = w.Write([]byte(msg))
		w.(http.Flusher).Flush()
	}
}

func (m *targetHandler) Timeout(d time.Duration, msg string) {
	m.writer = func(w http.ResponseWriter) {
		time.Sleep(d)
		w.Header()["Content-Length"] = []string{strconv.Itoa(len(msg))}
		_, _ = w.Write([]byte(msg))
		w.(http.Flusher).Flush()
	}
}

func (m *targetHandler) Close() {
	m.server.Close()
}

func newTargetHandler(msg string, tls bool) (string, *targetHandler) {
	m := targetHandler{}
	m.Msg(msg)
	m.server = httptest.NewUnstartedServer(&m)
	if tls {
		m.server.StartTLS()
	} else {
		m.server.Start()
	}
	log.Debugf("Started target site at %v", m.server.URL)
	return m.server.URL, &m
}

//
//
// Mock Redis reporter
//

type mockReporter struct {
	traffic map[string]*measured.Stats
	tmtx    sync.Mutex
}

func (mr *mockReporter) Report(ctx map[string]interface{}, stats *measured.Stats, deltaStats *measured.Stats, final bool) {
	mr.tmtx.Lock()
	defer mr.tmtx.Unlock()
	_deviceID := ctx[common.DeviceID]
	deviceID := ""
	if _deviceID != nil {
		deviceID = _deviceID.(string)
	}
	mr.traffic[deviceID] = stats
}
