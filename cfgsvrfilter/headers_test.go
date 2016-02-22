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
	fakeToken := "fake-token"
	dummyClientIP := "1.1.1.1"
	dummy := dummyHandler{}
	am, _ := New(&dummy, AuthToken(fakeToken), Domains([]string{"site1.com", "site2.org"}))
	req, _ := http.NewRequest("GET", "http://site1.com/abc.gz", nil)
	req.RemoteAddr = dummyClientIP
	am.ServeHTTP(nil, req)
	assert.Equal(t, fakeToken, dummy.req.Header.Get(cfgSvrAuthTokenHeader))
	assert.Equal(t, dummyClientIP, dummy.req.Header.Get(cfgSvrClientIPHeader))

	req, _ = http.NewRequest("GET", "http://site2.org/abc.gz", nil)
	req.RemoteAddr = dummyClientIP
	am.ServeHTTP(nil, req)
	assert.Equal(t, fakeToken, dummy.req.Header.Get(cfgSvrAuthTokenHeader))
	assert.Equal(t, dummyClientIP, dummy.req.Header.Get(cfgSvrClientIPHeader))

	req, _ = http.NewRequest("GET", "http://not-config-server.org/abc.gz", nil)
	req.RemoteAddr = dummyClientIP
	am.ServeHTTP(nil, req)
	assert.Equal(t, "", dummy.req.Header.Get(cfgSvrAuthTokenHeader))
	assert.Equal(t, "", dummy.req.Header.Get(cfgSvrClientIPHeader))
}

func TestPanics(t *testing.T) {
	dummy := dummyHandler{}
	assert.Panics(t, func() { _, _ = New(&dummy, AuthToken("fake-token")) },
		"should panic when no token provided")
	assert.Panics(t, func() { _, _ = New(&dummy, Domains([]string{"site1.com", "site2.org"})) },
		"should panic when no domains provided")
}
