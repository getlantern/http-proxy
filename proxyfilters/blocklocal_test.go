package proxyfilters

import (
	"net"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/getlantern/proxy/v3/filters"
)

type testResolver struct{}

func (r *testResolver) SplitHostPort(hostport string) (host string, port string, err error) {
	return net.SplitHostPort(hostport)
}
func (r *testResolver) ResolveIPAddr(network string, address string) (*net.IPAddr, error) {

	return &net.IPAddr{
		IP: net.IPv4(93, 184, 215, 16),
	}, nil
}

func TestBlockLocalBlocked(t *testing.T) {
	_, resp := doTestBlockLocal(t, []string{"localhost"}, "http://127.0.0.1/index.html")
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestBlockLocalPrivate(t *testing.T) {
	_, resp := doTestBlockLocal(t, []string{"localhost"}, "http://192.168.0.1/index.html")
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestBlockLocalException(t *testing.T) {
	_, resp := doTestBlockLocal(t, []string{"localhost"}, "http://localhost/index.html")
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestBlockLocalExceptionWithPort(t *testing.T) {
	_, resp := doTestBlockLocal(t, []string{"127.0.0.1:7300"}, "http://127.0.0.1:7300/index.html")
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestBlockLocalNotLocal(t *testing.T) {
	modifiedReq, resp := doTestBlockLocal(t, []string{"localhost"}, "http://example.com/index.html")
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	// assert that block local filter did the resolving and modified req.URL.Host
	exampleDotComIP := "93.184.215.16"
	assert.Equal(t, exampleDotComIP, modifiedReq.URL.Host)
}

func doTestBlockLocal(t *testing.T, exceptions []string, urlStr string) (*http.Request, *http.Response) {
	next := func(cs *filters.ConnectionState, req *http.Request) (*http.Response, *filters.ConnectionState, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
		}, cs, nil
	}

	filter := BlockLocal(exceptions, &testResolver{})
	req, _ := http.NewRequest(http.MethodGet, urlStr, nil)
	log.Debug(req.URL.Host)
	cs := filters.NewConnectionState(req, nil, nil)
	resp, _, _ := filter.Apply(cs, req, next)
	return req, resp
}
