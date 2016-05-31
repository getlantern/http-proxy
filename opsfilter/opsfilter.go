package opsfilter

import (
	"net"
	"net/http"

	"github.com/getlantern/golog"
	"github.com/getlantern/ops"

	"github.com/getlantern/http-proxy/filters"

	"github.com/getlantern/http-proxy-lantern/common"
)

var (
	log = golog.LoggerFor("logging")
)

type opsfilter struct{}

// New constructs a new filter that adds ops context.
func New() filters.Filter {
	return &opsfilter{}
}

func (f *opsfilter) Apply(resp http.ResponseWriter, req *http.Request, next filters.Next) error {
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
	return next()
}