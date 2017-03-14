package versioncheck

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	ip = "8.8.8.8"
)

func TestShouldRewrite(t *testing.T) {
	rewriteURL := "https://versioncheck.com/badversion"
	rewriteAddr := "versioncheck.com:443"
	f := New("3.1.1", rewriteURL, 1)
	req, _ := http.NewRequest("POST", "http://anysite.com", nil)
	assert.False(t, f.shouldRewrite(req), "should only redirect GET requests")
	req, _ = http.NewRequest("GET", "http://anysite.com", nil)
	assert.False(t, f.shouldRewrite(req), "should only redirect HTML requests")
	req.Header.Set("Accept", "text/html")
	assert.False(t, f.shouldRewrite(req), "should only redirect requests from browser")
	req.Header.Set("User-Agent", "Mozilla/5.0 xxx")
	assert.True(t, f.shouldRewrite(req), "should redirect if no version header present")
	req.Header.Set("X-Lantern-Version", "development")
	assert.True(t, f.shouldRewrite(req), "should redirect if the version is not semantic")
	req.Header.Set("X-Lantern-Version", "3.1.1")
	assert.False(t, f.shouldRewrite(req), "should not redirect if version equals to the min version")
	req.Header.Set("X-Lantern-Version", "3.1.2")
	assert.False(t, f.shouldRewrite(req), "should not redirect if version is above the min version")
	req.Header.Set("X-Lantern-Version", "3.11.0")
	assert.False(t, f.shouldRewrite(req), "should not redirect if version is above the min version")
	req.Header.Set("X-Lantern-Version", "3.1.0")
	assert.True(t, f.shouldRewrite(req), "should redirect if version is below the min version")

	f.RewriteIfNecessary(req)
	assert.Equal(t, rewriteURL, req.URL.String())
	assert.Equal(t, rewriteAddr, req.Host)
}

func TestPercentage(t *testing.T) {
	testPercentage(t, 1, true)
	testPercentage(t, 0.1, false)
	testPercentage(t, float64(1)/1000, false)
}

func testPercentage(t *testing.T, percentage float64, exact bool) {
	f := New("3.1.1", "http://versioncheck.com/badversion", percentage)
	req, _ := http.NewRequest("GET", "http://anysite.com", nil)
	req.Header.Set("Accept", "text/html")
	req.Header.Set("User-Agent", "Mozilla/5.0 xxx")
	hit := 0
	expected := int(percentage * oneMillion)
	for i := 0; i < oneMillion; i++ {
		if f.shouldRewrite(req) {
			hit++
		}
	}
	if exact {
		assert.Equal(t, expected, hit)
	} else {
		assert.InDelta(t, expected, hit, float64(expected)/10)
	}
}
