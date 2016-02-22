package configserverfilter

import (
	"net/http"
	"testing"

	"github.com/getlantern/testify/assert"

	"github.com/getlantern/http-proxy-lantern/common"
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
	assert.Equal(t, fakeToken, dummy.req.Header.Get(common.CfgSvrAuthTokenHeader))
	assert.Equal(t, dummyClientIP, dummy.req.Header.Get(common.CfgSvrClientIPHeader))

	req, _ = http.NewRequest("GET", "http://site2.org/abc.gz", nil)
	req.RemoteAddr = dummyClientIP
	am.ServeHTTP(nil, req)
	assert.Equal(t, fakeToken, dummy.req.Header.Get(common.CfgSvrAuthTokenHeader))
	assert.Equal(t, dummyClientIP, dummy.req.Header.Get(common.CfgSvrClientIPHeader))

	req, _ = http.NewRequest("GET", "http://not-config-server.org/abc.gz", nil)
	req.RemoteAddr = dummyClientIP
	am.ServeHTTP(nil, req)
	assert.Equal(t, "", dummy.req.Header.Get(common.CfgSvrAuthTokenHeader))
	assert.Equal(t, "", dummy.req.Header.Get(common.CfgSvrClientIPHeader))
}

func TestInitialize(t *testing.T) {
	dummy := dummyHandler{}
	_, err := New(&dummy, AuthToken("fake-token"))
	assert.Error(t, err, "should return error when no token provided")
	_, err = New(&dummy, Domains([]string{"site1.com", "site2.org"}))
	assert.Error(t, err, "should return error when no domains provided")
}
