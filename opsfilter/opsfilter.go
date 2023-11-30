package opsfilter

import (
	"net"
	"net/http"
	"strings"

	"github.com/getlantern/netx"
	"github.com/getlantern/proxy/v3/filters"

	"github.com/getlantern/http-proxy-lantern/v2/common"
	"github.com/getlantern/http-proxy-lantern/v2/listeners"
	"github.com/getlantern/http-proxy-lantern/v2/tlslistener"
)

type opsfilter struct {
}

// New constructs a new filter that adds ops context.
func New() filters.Filter {
	return &opsfilter{}
}

func (f *opsfilter) Apply(cs *filters.ConnectionState, req *http.Request, next filters.Next) (*http.Response, *filters.ConnectionState, error) {
	originHost, originPort, _ := net.SplitHostPort(req.Host)
	if (originPort == "0" || originPort == "") && req.Method != http.MethodConnect {
		// Default port for HTTP
		originPort = "80"
	}
	if originHost == "" && !strings.Contains(req.Host, ":") {
		originHost = req.Host
	}

	clientIP, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		clientIP = req.RemoteAddr
	}

	measuredCtx := map[string]interface{}{
		"origin":          req.Host,
		common.OriginHost: originHost,
		common.OriginPort: originPort,
		common.ClientIP:   clientIP,
	}

	addVal := func(ctxKey string, val interface{}) {
		if val != nil && val != "" {
			measuredCtx[ctxKey] = val
		}
	}

	addArrayHeader := func(ctxKey, headerKey string) {
		vals := req.Header.Values(headerKey)
		if len(vals) == 0 {
			return
		}
		addVal(ctxKey, vals)
	}

	addStringHeader := func(ctxKey, headerKey string) {
		val := req.Header.Get(headerKey)
		addVal(ctxKey, val)
	}

	// Note we changed this with flashlight version v7.6.24 or 25 to explicity
	// report the application version separately. Starting in early 2023,
	// we began reporting the library (flashlight) version in X-Lantern-Version.
	// Prior to that, we were reporting the version of the application.
	addStringHeader(common.LibraryVersion, common.LibraryVersionHeader)

	// On persistent HTTP connections, some or all of the below may be missing on requests after the first. By only setting
	// the values when they're available, the measured listener will preserve any values that were already included in the
	// first request on the connection.
	addStringHeader(common.DeviceID, common.DeviceIdHeader)
	addStringHeader(common.KernelArch, common.KernelArchHeader)
	addStringHeader(common.AppVersion, common.AppVersionHeader)
	addStringHeader(common.Platform, common.PlatformHeader)
	addStringHeader(common.App, common.AppHeader)
	addStringHeader(common.Locale, common.LocaleHeader)
	addStringHeader(common.TimeZone, common.TimeZoneHeader)
	addArrayHeader(common.SupportedDataCaps, common.SupportedDataCapsHeader)

	netx.WalkWrapped(cs.Downstream(), func(conn net.Conn) bool {
		pdc, ok := conn.(tlslistener.ProbingDetectingConn)
		if ok {
			addVal(common.ProbingError, pdc.ProbingError())
			return false
		}
		return true
	})

	// Send the same context data to measured as well
	wc := cs.Downstream().(listeners.WrapConn)
	wc.ControlMessage("measured", measuredCtx)

	return next(cs, req)
}
