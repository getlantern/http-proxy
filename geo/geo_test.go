package geo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLookup(t *testing.T) {
	l := New(2)
	doTestLookup(t, l, "188.166.36.215", "nl", true, 1, 0, 1, 0)
	doTestLookup(t, l, "188.166.36.215", "nl", true, 1, 1, 1, 0)
	doTestLookup(t, l, "139.59.59.44", "in", true, 2, 1, 2, 0)
	doTestLookup(t, l, "139.59.59.44", "in", true, 2, 2, 2, 0)
	doTestLookup(t, l, "45.55.177.174", "us", true, 2, 2, 3, 0)
	doTestLookup(t, l, "139.59.59.44", "in", true, 2, 3, 3, 0)
	doTestLookup(t, l, "188.166.36.215", "nl", true, 2, 3, 4, 0)
	doTestLookup(t, l, "adsfs423afsd234:2343", "", false, 2, 3, 5, 1)
}

func doTestLookup(t *testing.T, l Lookup, ip string, expectedCountry string, expectNoError bool, expectedCacheSize int, expectedCacheHits int, expectedNetworkLookups int, expectedNetworkLookupErrors int) {
	country, err := l.Country(ip)
	if expectNoError {
		assert.NoError(t, err)
	} else {
		assert.Error(t, err)
	}
	assert.Equal(t, expectedCountry, country)
	assert.Equal(t, expectedCacheSize, l.CacheSize())
	assert.Equal(t, expectedCacheHits, l.CacheHits())
	assert.Equal(t, expectedNetworkLookups, l.NetworkLookups())
	assert.Equal(t, expectedNetworkLookupErrors, l.NetworkLookupErrors())
}
