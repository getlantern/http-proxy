package opsfilter

import (
	"net"
	"net/http"
	"strings"

	"github.com/getlantern/golog"
	"github.com/getlantern/ops"
	"github.com/getlantern/proxy/filters"

	"github.com/getlantern/http-proxy-lantern/bbr"
	"github.com/getlantern/http-proxy-lantern/common"
	"github.com/getlantern/http-proxy/listeners"
)

var (
	log = golog.LoggerFor("logging")
)

type opsfilter struct {
	bm bbr.Middleware
}

// New constructs a new filter that adds ops context.
func New(bm bbr.Middleware) filters.Filter {
	return &opsfilter{bm}
}

func (f *opsfilter) Apply(ctx filters.Context, req *http.Request, next filters.Next) (*http.Response, filters.Context, error) {
	deviceID := req.Header.Get(common.DeviceIdHeader)
	originHost, originPort, _ := net.SplitHostPort(req.Host)
	if (originPort == "0" || originPort == "") && req.Method != http.MethodConnect {
		// Default port for HTTP
		originPort = "80"
	}
	if originHost == "" && !strings.Contains(req.Host, ":") {
		originHost = req.Host
	}

	op := ops.Begin("proxy").
		Set("device_id", deviceID).
		Set("origin", req.Host).
		Set("origin_host", originHost).
		Set("origin_port", originPort)
	log.Debug("Starting op")
	defer op.End()

	opsCtx := map[string]interface{}{
		"deviceid":    deviceID,
		"origin":      req.Host,
		"origin_host": originHost,
		"origin_port": originPort,
	}

	clientIP, _, err := net.SplitHostPort(req.RemoteAddr)
	if err == nil {
		op.Set("client_ip", clientIP)
		opsCtx["client_ip"] = clientIP
	}

	// Send the same context data to measured as well
	wc := ctx.DownstreamConn().(listeners.WrapConn)
	wc.ControlMessage("measured", opsCtx)

	resp, nextCtx, nextErr := next(ctx, req)
	op.FailIf(nextErr)

	return resp, nextCtx, nextErr
}
