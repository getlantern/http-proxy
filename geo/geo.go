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
// statistics about itself.
type Lookup interface {
	// Country looks up the 2 digit ISO 3166 country code for the given IP address
	// and returns the address or an error if there was an error looking it up.
	Country(ip string) (string, error)

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

func New(maxSize int) Lookup {
	cache, _ := lru.New(maxSize)
	return &lookup{cache: cache}
}

func (l *lookup) Country(ip string) (string, error) {
	cached, found := l.cache.Get(ip)
	if found {
		atomic.AddInt64(&l.cacheHits, 1)
		return cached.(string), nil
	}
	lookedUp, _, err := geolookup.LookupIP(ip, rt)
	atomic.AddInt64(&l.networkLookups, 1)
	if err != nil {
		atomic.AddInt64(&l.networkLookupErrors, 1)
		return "", err
	}
	country := strings.ToLower(lookedUp.Country.IsoCode)
	l.cache.Add(ip, country)
	return country, nil
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
