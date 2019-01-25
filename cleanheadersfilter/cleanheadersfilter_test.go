package cleanheadersfilter

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWhitelisted(t *testing.T) {
	filter := New().(*filter)
	req := buildRequest("https://getlantern.org/stuff")
	filter.stripHeadersIfNecessary(req)
	assert.Equal(t, "A", req.Header.Get("X-Lantern-A"))
	assert.Equal(t, "B", req.Header.Get("X-Lantern-B"))
	assert.Equal(t, "close", req.Header.Get("Connection"))
	assert.Empty(t, req.Header.Get("Proxy-Connection"))
	assert.Equal(t, "O", req.Header.Get("Other"))
}

func TestNotWhitelisted(t *testing.T) {
	filter := New().(*filter)
	req := buildRequest("https://alipay.com/stuff")
	filter.stripHeadersIfNecessary(req)
	assert.Empty(t, req.Header.Get("X-Lantern-A"))
	assert.Empty(t, req.Header.Get("X-Lantern-B"))
	assert.Equal(t, "close", req.Header.Get("Connection"))
	assert.Empty(t, req.Header.Get("Proxy-Connection"))
	assert.Equal(t, "O", req.Header.Get("Other"))
}

func buildRequest(url string) *http.Request {
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("X-Lantern-A", "A")
	req.Header.Set("x-lantern-b", "B")
	req.Header.Set("proxy-Connection", "close")
	req.Header.Set("Other", "O")
	return req
}
