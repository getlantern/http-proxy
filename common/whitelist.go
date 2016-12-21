package common

import (
	"net"
	"net/http"
	"strings"

	"github.com/getlantern/golog"
)

var (
	log = golog.LoggerFor("common")
)

// Whitelist contains domains we should whitelist both for throttling and for
// not counting towards the bandwidth cap.
type Whitelist struct {
	Domains []string
}

// NewRawWhitelist creates a new Whitelist with the given whitelisted domains,
// separated by commas.
func NewRawWhitelist(raw string) *Whitelist {
	domains := strings.Split(raw, ",")
	// Allow whitespace from the command line.
	for i, d := range domains {
		domains[i] = strings.TrimSpace(d)
	}
	return NewWhitelist(domains)
}

// NewWhitelist creates a new Whitelist with the given whitelisted domains.
func NewWhitelist(domains []string) *Whitelist {
	return &Whitelist{Domains: domains}
}

// Whitelisted returns whether or not the given request should be whitelisted.
func (f *Whitelist) Whitelisted(req *http.Request) bool {
	origin, _, err := net.SplitHostPort(req.Host)
	if err != nil {
		origin = req.Host
	}
	for _, d := range f.Domains {
		if origin == d {
			return true
		}
	}
	return false
}
