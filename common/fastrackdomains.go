package common

import (
	"net/http"
	"strings"

	"github.com/getlantern/golog"
)

var (
	log = golog.LoggerFor("common")
)

// FasttrackDomains contains domains we should whitelist both for throttling and for
// not counting towards the bandwidth cap.
type FasttrackDomains struct {
	Domains []string
}

// NewRawFasttrackDomains creates a new DomainList with the given whitelisted domains,
// separated by commas.
func NewRawFasttrackDomains(raw string) *FasttrackDomains {
	doms := strings.Split(raw, ",")
	domains := make([]string, 0)
	// Allow whitespace from the command line.
	for _, dom := range doms {
		d := strings.TrimSpace(dom)
		if d != "" {
			domains = append(domains, d)
		}
	}
	return NewFasttrackDomains(domains)
}

// NewFasttrackDomains creates a new Whitelist with the given whitelisted domains.
func NewFasttrackDomains(domains []string) *FasttrackDomains {
	return &FasttrackDomains{Domains: domains}
}

// Whitelisted returns whether or not the given request should be whitelisted.
func (f *FasttrackDomains) Whitelisted(req *http.Request) bool {
	for _, d := range f.Domains {
		if strings.HasPrefix(req.Host, d) {
			return true
		}
	}
	return false
}
