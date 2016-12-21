package common

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWhitelist(t *testing.T) {

	domains := "getiantem.org, lantern-pro-server.herokuapp.com"

	wl := NewRawDomainList(domains)
	req, _ := http.NewRequest("GET", "http://site1.com:80/abc.gz", nil)
	assert.False(t, wl.Whitelisted(req), "http://site1.com:80/abc.gz should not be whitelisted")

	req, _ = http.NewRequest("GET", "http://getiantem.org", nil)
	assert.True(t, wl.Whitelisted(req))

	req, _ = http.NewRequest("GET", "http://lantern-pro-server.herokuapp.com", nil)
	assert.True(t, wl.Whitelisted(req), "lantern-pro-server.herokuapp.com not whitelisted?")

	// Just make sure it accepts the empty string.
	wl = NewRawDomainList("")
	req, _ = http.NewRequest("GET", "http://getiantem.org", nil)
	assert.False(t, wl.Whitelisted(req), "http://getiantem.org should not be whitelisted")
}
