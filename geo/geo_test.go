package geo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLookup(t *testing.T) {
	l := New(2)
	doTestLookup(t, l, "188.166.36.215", "nl", 1, 0, 1, 0)
	doTestLookup(t, l, "188.166.36.215", "nl", 1, 1, 1, 0)
	doTestLookup(t, l, "139.59.59.44", "in", 2, 1, 2, 0)
	doTestLookup(t, l, "139.59.59.44", "in", 2, 2, 2, 0)
	doTestLookup(t, l, "45.55.177.174", "us", 2, 2, 3, 0)
	doTestLookup(t, l, "139.59.59.44", "in", 2, 3, 3, 0)
	doTestLookup(t, l, "188.166.36.215", "nl", 2, 3, 4, 0)
	doTestLookup(t, l, "adsfs423afsd234:2343", "", 2, 3, 5, 1)
	doTestLookup(t, l, "adsfs423afsd234:2343", "", 2, 4, 5, 1)
}

func doTestLookup(t *testing.T, l Lookup, ip string, expectedCountry string, expectedCacheSize int, expectedCacheHits int, expectedNetworkLookups int, expectedNetworkLookupErrors int) {
	country := l.CountryCode(ip)
	assert.Equal(t, expectedCountry, country)
	assert.Equal(t, expectedCacheSize, l.CacheSize())
	assert.Equal(t, expectedCacheHits, l.CacheHits())
	assert.Equal(t, expectedNetworkLookups, l.NetworkLookups())
	assert.Equal(t, expectedNetworkLookupErrors, l.NetworkLookupErrors())
}
