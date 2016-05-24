package opsfilter

import (
	"net"
	"net/http"

	"github.com/getlantern/golog"
	"github.com/getlantern/ops"

	"github.com/getlantern/http-proxy-lantern/common"
)

var (
	log = golog.LoggerFor("logging")
)

type filter struct {
	next http.Handler
}

// New constructs a new filter that adds ops context.
func New(next http.Handler) http.Handler {
	return &filter{next}
}

func (f *filter) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	deviceID := req.Header.Get(common.DeviceIdHeader)
	op := ops.
		Enter("proxy").
		Put("device_id", deviceID).
		Put("origin", req.Host)
	defer op.Exit()
	clientIP, _, err := net.SplitHostPort(req.RemoteAddr)
	if err == nil {
		op.Put("client_ip", clientIP)
	}
	f.next.ServeHTTP(resp, req)
}
