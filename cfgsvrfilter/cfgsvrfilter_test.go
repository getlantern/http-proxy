package cfgsvrfilter

import (
	"net/http"
	"testing"

	"github.com/getlantern/testify/assert"
)

type dummyHandler struct{ req *http.Request }

func (h *dummyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.req = r
}

func TestAttachAuthToken(t *testing.T) {
	dummy := dummyHandler{}
	am, _ := New(&dummy, AuthToken("fake-token"), Domains([]string{"site1.com", "site2.org"}))
	req, _ := http.NewRequest("GET", "http://site1.com/abc.gz", nil)
	am.ServeHTTP(nil, req)
	assert.Equal(t, "fake-token", dummy.req.Header.Get(cfgSvrAuthTokenHeader))

	req, _ = http.NewRequest("GET", "http://site2.org/abc.gz", nil)
	am.ServeHTTP(nil, req)
	assert.Equal(t, "fake-token", dummy.req.Header.Get(cfgSvrAuthTokenHeader))

	req, _ = http.NewRequest("GET", "http://not-config-server.org/abc.gz", nil)
	am.ServeHTTP(nil, req)
	assert.Equal(t, "", dummy.req.Header.Get(cfgSvrAuthTokenHeader))
}

func TestPanics(t *testing.T) {
	dummy := dummyHandler{}
	assert.Panics(t, func() { _, _ = New(&dummy, AuthToken("fake-token")) },
		"should panic when no token provided")
	assert.Panics(t, func() { _, _ = New(&dummy, Domains([]string{"site1.com", "site2.org"})) },
		"should panic when no domains provided")
}
