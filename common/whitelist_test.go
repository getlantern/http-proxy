package common

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWhitelist(t *testing.T) {

	domains := []string{"getiantem.org", "lantern-pro-server.herokuapp.com"}

	wl := NewWhitelist(domains)
	req, _ := http.NewRequest("GET", "http://site1.com:80/abc.gz", nil)
	assert.False(t, wl.Whitelisted(req))

	req, _ = http.NewRequest("GET", "http://getiantem.org", nil)
	assert.True(t, wl.Whitelisted(req))
}
