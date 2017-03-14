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
	return &VersionChecker{minVersion, semver.MustParse(minVersion), u, rewriteAddr, int(percentage * oneMillion)}
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
	c.RewriteIfNecessary(req)
	return next()
}

func (c *VersionChecker) RewriteIfNecessary(req *http.Request) {
	if c.shouldRewrite(req) {
		req.URL = c.rewriteURL
		req.Host = c.rewriteAddr
	}
}

func (c *VersionChecker) shouldRewrite(req *http.Request) bool {
	// the first request from browser should always be GET
	if req.Method != http.MethodGet {
		return false
	}
	// typical browsers always have this as the first value
	if req.Header.Get("Accept") != "text/html" {
		return false
	}
	// This covers almost all browsers
	if !strings.HasPrefix(req.Header.Get("User-Agent"), "Mozilla/") {
		return false
	}
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
