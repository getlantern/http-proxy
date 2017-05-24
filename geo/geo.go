// Package geo provides functionality for looking up country codes based on
// IP addresses.
package geo

import (
	"net/http"
	"strings"
	"sync/atomic"

	"github.com/getlantern/geolookup"
	"github.com/getlantern/golog"
	"github.com/hashicorp/golang-lru"
)

var (
	log = golog.LoggerFor("http-proxy-lantern.geo")

	rt = &http.Transport{}
)

// Lookup allows looking up the country for an IP address and exposes some
// statistics about itself. It caches results indefinitely, up to a configurable
// cache size.
type Lookup interface {
	// CountryCode looks up the 2 digit ISO 3166 country code for the given IP
	// address and returns "" if there was an error looking up the country.
	CountryCode(ip string) string

	// CacheSize returns the current size of the cache.
	CacheSize() int

	// CacheHits counts the number of lookups that were performed from cache.
	CacheHits() int

	// NetworkLookups counts the number of lookups that were performed over the
	// network (whether successful or not).
	NetworkLookups() int

	// NetworkLookupErrors counts the number of errors encountered looking up ips
	// over the network.
	NetworkLookupErrors() int
}

type lookup struct {
	cache               *lru.Cache
	cacheHits           int64
	networkLookups      int64
	networkLookupErrors int64
}

// New constructs a new caching Lookup with an LRU cache limited to maxSize.
func New(maxSize int) Lookup {
	cache, _ := lru.New(maxSize)
	return &lookup{cache: cache}
}

func (l *lookup) CountryCode(ip string) string {
	cached, found := l.cache.Get(ip)
	if found {
		atomic.AddInt64(&l.cacheHits, 1)
		return cached.(string)
	}
	lookedUp, _, err := geolookup.LookupIP(ip, rt)
	atomic.AddInt64(&l.networkLookups, 1)
	if err != nil {
		atomic.AddInt64(&l.networkLookupErrors, 1)
		log.Errorf("Error looking up country for %v: %v", ip, err)
		l.cache.Add(ip, "")
		return ""
	}
	country := strings.ToLower(lookedUp.Country.IsoCode)
	l.cache.Add(ip, country)
	return country
}

func (l *lookup) CacheSize() int {
	return l.cache.Len()
}

func (l *lookup) CacheHits() int {
	return int(atomic.LoadInt64(&l.cacheHits))
}

func (l *lookup) NetworkLookups() int {
	return int(atomic.LoadInt64(&l.networkLookups))
}

func (l *lookup) NetworkLookupErrors() int {
	return int(atomic.LoadInt64(&l.networkLookupErrors))
}
