// package versioncheck checks the GET requests from browsers to see if the
// X-Lantern-Version header is absent or below than a semantic version,
// redirects a fraction of the requests to a predefined URL. The purpose is to
// show an upgrade notice to the users with outdated Lantern client.
package versioncheck

import (
	"math/rand"
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

type filter struct {
	minVersionString string
	minVersion       semver.Version
	redirectURL      string
	redirectHost     string
	ppm              int
}

// New constructs a new filter to check the request and redirect if required.
// It panics if the minVersion string is not semantic versioned, or the
// redirect URL is malformed.
func New(minVersion string, redirectURL string, percentage float64) filters.Filter {
	u, err := url.Parse(redirectURL)
	if err != nil {
		panic(err)
	}
	return &filter{minVersion, semver.MustParse(minVersion), redirectURL, u.Host, int(percentage * oneMillion)}
}

func (f *filter) Apply(resp http.ResponseWriter, req *http.Request, next filters.Next) error {
	if f.shouldRedirect(req) {
		http.Redirect(resp, req, f.redirectURL, http.StatusTemporaryRedirect)
		return nil
	}
	return next()
}

func (f *filter) shouldRedirect(req *http.Request) bool {
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
	if req.Host == f.redirectHost {
		return false
	}
	version := req.Header.Get(common.VersionHeader)
	v, e := semver.Make(version)
	if e == nil && v.GTE(f.minVersion) {
		return false
	}
	if random.Intn(oneMillion) >= f.ppm {
		return false
	}
	return true
}
