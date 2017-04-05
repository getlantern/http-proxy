// package versioncheck checks the GET requests from browsers to see if the
// X-Lantern-Version header is absent or below than a semantic version,
// rewrites a fraction of the requests to a predefined URL. The purpose is to
// show an upgrade notice to the users with outdated Lantern client.
package versioncheck

import (
	"crypto/tls"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/blang/semver"
	"github.com/getlantern/golog"

	"github.com/getlantern/http-proxy-lantern/common"
	"github.com/getlantern/http-proxy/filters"
)

var (
	log = golog.LoggerFor("versioncheck")

	random = rand.New(rand.NewSource(time.Now().UnixNano()))
)

const (
	oneMillion = 100 * 100 * 100
)

type VersionChecker struct {
	minVersionString string
	minVersion       semver.Version
	rewriteURL       *url.URL
	rewriteURLString string
	rewriteAddr      string
	ppm              int
}

// New constructs a VersionChecker to check the request and rewrite if required.
// It panics if the minVersion string is not semantic versioned, or the
// rewrite URL is malformed.
func New(minVersion string, rewriteURL string, percentage float64) *VersionChecker {
	u, err := url.Parse(rewriteURL)
	if err != nil {
		panic(err)
	}
	rewriteAddr := u.Host

	if u.Scheme == "https" {
		rewriteAddr = rewriteAddr + ":443"
	}
	return &VersionChecker{minVersion, semver.MustParse(minVersion), u, rewriteURL, rewriteAddr, int(percentage * oneMillion)}
}

type Dial func(network, address string) (net.Conn, error)

// Dialer wraps Dial to dial TLS when the requested host matchs the host in
// rewriteURL. If the rewriteURL is not https, it returns Dial as is.
func (c *VersionChecker) Dialer(d Dial) Dial {
	if c.rewriteURL.Scheme != "https" {
		return d
	}
	return func(network, address string) (net.Conn, error) {
		conn, err := d(network, address)
		if err != nil {
			return conn, err
		}
		if c.rewriteAddr == address {
			conn = tls.Client(conn, &tls.Config{ServerName: c.rewriteURL.Host})
		}
		return conn, err
	}
}

func (c *VersionChecker) Filter() filters.Filter {
	return c
}

func (c *VersionChecker) Apply(resp http.ResponseWriter, req *http.Request, next filters.Next) error {
	if req.Method == http.MethodConnect {
		if c.redirectConnectIfNecessary(resp, req) {
			// stop here as the response has already been written
			return nil
		}
		return next()
	}
	c.RewriteIfNecessary(req)
	return next()
}

func (c *VersionChecker) RewriteIfNecessary(req *http.Request) {
	defer req.Header.Del(common.VersionHeader)
	if !c.shouldRewrite(req) {
		return
	}
	log.Debugf("Rewriting %s://%s%s to %s%s",
		req.Method,
		req.Host,
		req.URL.Path,
		c.rewriteAddr,
		c.rewriteURL.Path,
	)
	req.URL = c.rewriteURL
	req.Host = c.rewriteAddr
}

func (c *VersionChecker) shouldRewrite(req *http.Request) bool {
	// the first request from browser should always be GET
	if req.Method != http.MethodGet {
		return false
	}
	// typical browsers always have this as the first value
	if !strings.HasPrefix(req.Header.Get("Accept"), "text/html") {
		return false
	}
	// This covers almost all browsers
	if !strings.HasPrefix(req.Header.Get("User-Agent"), "Mozilla/") {
		return false
	}
	return c.matchVersion(req)
}

func (c *VersionChecker) redirectConnectIfNecessary(w http.ResponseWriter, req *http.Request) bool {
	if !c.matchVersion(req) {
		return false
	}
	_, port, err := net.SplitHostPort(req.Host)
	if err != nil || port != "80" {
		return false
	}
	h, ok := w.(http.Hijacker)
	if !ok {
		return false
	}
	conn, _, err := h.Hijack()
	if err != nil {
		// If there's an error hijacking, it's because the connection has already
		// been hijacked (a programming error). Not much we can do other than
		// report an error.
		log.Errorf("Unable to hijack connection: %s", err)
		return true
	}
	defer conn.Close()

	log.Debugf("Redirecting %s://%s%s to %s",
		req.Method,
		req.Host,
		req.URL.Path,
		c.rewriteURLString,
	)
	// Acknowledge the CONNECT request
	resp := &http.Response{
		StatusCode: http.StatusOK,
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     w.Header(),
	}
	if err := resp.Write(conn); err != nil {
		log.Debugf("error write: %v", err)
		return true
	}

	// Make sure the application sent something and started waiting for the
	// response.
	var buf [1]byte
	_, _ = conn.Read(buf[:])

	// Send the actual response to application.
	resp = &http.Response{
		StatusCode: http.StatusFound,
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header: http.Header{
			"Location": []string{c.rewriteURLString},
		},
		Close: true,
	}
	if err := resp.Write(conn); err != nil {
		log.Debugf("error write: %v", err)
	}
	return true
}

func (c *VersionChecker) matchVersion(req *http.Request) bool {
	// Avoid infinite loop
	if req.Host == c.rewriteURL.Host {
		return false
	}
	version := req.Header.Get(common.VersionHeader)
	v, e := semver.Make(version)
	if e == nil && v.GTE(c.minVersion) {
		return false
	}
	if random.Intn(oneMillion) >= c.ppm {
		return false
	}
	return true
}
