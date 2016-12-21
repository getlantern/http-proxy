package common

import (
	"net"
	"net/http"

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

// NewWhitelist creates a new Whitelist with the given whitelisted domains.
func NewWhitelist(domains []string) *Whitelist {
	return &Whitelist{Domains: domains}
}

// Whitelisted returns whether or not the given request should be whitelisted.
func (f *Whitelist) Whitelisted(req *http.Request) bool {
	origin, _, err := net.SplitHostPort(req.Host)
	if err != nil {
		log.Debugf("Got error for host: %v", req.Host)
		origin = req.Host
	}
	for _, d := range f.Domains {
		if origin == d {
			return true
		}
	}
	return false
}
