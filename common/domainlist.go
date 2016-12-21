package common

import (
	"net/http"
	"strings"

	"github.com/getlantern/golog"
)

var (
	log = golog.LoggerFor("common")
)

// DomainList contains domains we should whitelist both for throttling and for
// not counting towards the bandwidth cap.
type DomainList struct {
	Domains []string
}

// NewRawDomainList creates a new DomainList with the given whitelisted domains,
// separated by commas.
func NewRawDomainList(raw string) *DomainList {
	doms := strings.Split(raw, ",")
	domains := make([]string, 0)
	// Allow whitespace from the command line.
	for _, dom := range doms {
		d := strings.TrimSpace(dom)
		if d != "" {
			domains = append(domains, d)
		}
	}
	return NewDomainList(domains)
}

// NewDomainList creates a new Whitelist with the given whitelisted domains.
func NewDomainList(domains []string) *DomainList {
	return &DomainList{Domains: domains}
}

// Whitelisted returns whether or not the given request should be whitelisted.
func (f *DomainList) Whitelisted(req *http.Request) bool {
	for _, d := range f.Domains {
		if d == req.Host {
			return true
		}
	}
	return false
}
