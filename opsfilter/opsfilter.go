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
	version := req.Header.Get(common.LegacyVersionHeader)
	if version == "" {
		// If there's no legacy version header, set the "version" to the
		// library version to be consistent with more recent lantern
		// versions post early 2023.
		version = req.Header.Get(common.LibraryVersionHeader)
	}

	clientIP, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		clientIP = req.RemoteAddr
	}

	measuredCtx := map[string]interface{}{
		"origin":          req.Host,
		common.OriginHost: originHost,
		common.OriginPort: originPort,

		// Note we changed this with flashlight version v7.6.24 to explicity
		// report the application version and the library version separately.
		// Prior to that, for about 6 months, we were reporting the library
		// version. Prior to that (around March of 2023), we were reporting
		// the version of the application.
		common.Version:  version,
		common.ClientIP: clientIP,
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

	addStringHeader := func(ctxKey, headerKey string) {
		val := req.Header.Get(headerKey)
		if val != "" {
			measuredCtx[ctxKey] = val
		}
	}

	// On persistent HTTP connections, some or all of the below may be missing on requests after the first. By only setting
	// the values when they're available, the measured listener will preserve any values that were already included in the
	// first request on the connection.
	addStringHeader(common.DeviceID, common.DeviceIdHeader)
	addStringHeader(common.AppVersion, common.AppVersionHeader)
	addStringHeader(common.LibraryVersion, common.LibraryVersionHeader)
	addStringHeader(common.Platform, common.PlatformHeader)
	addStringHeader(common.App, common.AppHeader)
	addStringHeader(common.Locale, common.LocaleHeader)
	addStringHeader(common.TimeZone, common.TimeZoneHeader)
	addMeasuredHeader(common.SupportDataCaps, req.Header[common.SupportedDataCapsHeader])

	netx.WalkWrapped(cs.Downstream(), func(conn net.Conn) bool {
		pdc, ok := conn.(tlslistener.ProbingDetectingConn)
		if ok {
			addMeasuredHeader(common.ProbingError, pdc.ProbingError())
			return false
		}
		return true
	})

	// Send the same context data to measured as well
	wc := cs.Downstream().(listeners.WrapConn)
	wc.ControlMessage("measured", measuredCtx)

	return next(cs, req)
}
