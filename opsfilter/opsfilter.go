package opsfilter

import (
	"net"
	"net/http"
	"strings"

	"github.com/getlantern/golog"
	"github.com/getlantern/netx"
	"github.com/getlantern/proxy/v2/filters"

	"github.com/getlantern/http-proxy-lantern/v2/common"
	"github.com/getlantern/http-proxy-lantern/v2/tlslistener"
	"github.com/getlantern/http-proxy/listeners"
)

var (
	log = golog.LoggerFor("logging")
)

type opsfilter struct {
}

// New constructs a new filter that adds ops context.
func New() filters.Filter {
	return &opsfilter{}
}

func (f *opsfilter) Apply(cs *filters.ConnectionState, req *http.Request, next filters.Next) (*http.Response, *filters.ConnectionState, error) {
	deviceID := req.Header.Get(common.DeviceIdHeader)
	originHost, originPort, _ := net.SplitHostPort(req.Host)
	if (originPort == "0" || originPort == "") && req.Method != http.MethodConnect {
		// Default port for HTTP
		originPort = "80"
	}
	if originHost == "" && !strings.Contains(req.Host, ":") {
		originHost = req.Host
	}
	platform := req.Header.Get(common.PlatformHeader)
	version := req.Header.Get(common.VersionHeader)
	app := req.Header.Get(common.AppHeader)
	locale := req.Header.Get(common.LocaleHeader)

	measuredCtx := map[string]interface{}{
		"origin":      req.Host,
		"origin_host": originHost,
		"origin_port": originPort,
	}

	addMeasuredHeader := func(key string, headerValue interface{}) {
		if headerValue != nil && headerValue != "" {
			headerArray, ok := headerValue.([]string)
			if ok && len(headerArray) == 0 {
				return
			}
			measuredCtx[key] = headerValue
		}
	}

	// On persistent HTTP connections, some or all of the below may be missing on requests after the first. By only setting
	// the values when they're available, the measured listener will preserve any values that were already included in the
	// first request on the connection.
	addMeasuredHeader("deviceid", deviceID)
	addMeasuredHeader("app_version", version)
	addMeasuredHeader("app_platform", platform)
	addMeasuredHeader("app", app)
	addMeasuredHeader("locale", locale)
	addMeasuredHeader("supported_data_caps", req.Header[common.SupportedDataCaps])
	addMeasuredHeader("time_zone", req.Header.Get(common.TimeZoneHeader))

	netx.WalkWrapped(cs.Downstream(), func(conn net.Conn) bool {
		pdc, ok := conn.(tlslistener.ProbingDetectingConn)
		if ok {
			addMeasuredHeader("probing_error", pdc.ProbingError())
			return false
		}
		return true
	})

	clientIP, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		clientIP = req.RemoteAddr
	}
	measuredCtx["client_ip"] = clientIP

	// Send the same context data to measured as well
	wc := cs.Downstream().(listeners.WrapConn)
	wc.ControlMessage("measured", measuredCtx)

	return next(cs, req)
}
