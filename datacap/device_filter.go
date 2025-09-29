package datacap

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"sync"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/getlantern/http-proxy-lantern/v2/common"
	"github.com/getlantern/http-proxy-lantern/v2/domains"
	"github.com/getlantern/http-proxy-lantern/v2/instrument"
	"github.com/getlantern/http-proxy-lantern/v2/listeners"
	"github.com/getlantern/http-proxy-lantern/v2/usage"
	"github.com/getlantern/proxy/v3/filters"
)

var (
	epoch = time.Date(2016, 1, 1, 0, 0, 0, 0, time.UTC)

	alwaysThrottle = listeners.NewRateLimiter(10, 10) // this is basically unusably slow, only used for malicious or really old/broken clients

	defaultThrottleRate = int64(5000 * 1024 / 8) // 5 Mbps
)

// deviceFilter handles filtering and throttling of requests based on datacap
type deviceFilter struct {
	datacapClient      DatacapSidecarClient
	instrument         instrument.Instrument
	sendXBQHeader      bool
	limitersByDevice   map[string]*listeners.RateLimiter
	limitersByDeviceMx sync.Mutex
}

// Settings represents the datacap settings for a device
type Settings struct {
	Threshold int64
}

// NewFilter creates a new datacap filter
func NewFilter(datacapClient DatacapSidecarClient, instrument instrument.Instrument, sendXBQHeader bool) *deviceFilter {
	return &deviceFilter{
		datacapClient:    datacapClient,
		instrument:       instrument,
		sendXBQHeader:    sendXBQHeader,
		limitersByDevice: make(map[string]*listeners.RateLimiter, 0),
	}
}

// Apply applies the datacap filter to the request
func (f *deviceFilter) Apply(cs *filters.ConnectionState, req *http.Request, next filters.Next) (*http.Response, *filters.ConnectionState, error) {

	if log.IsTraceEnabled() {
		reqStr, _ := httputil.DumpRequest(req, true)
		log.Tracef("DeviceFilter Middleware received request:\n%s", reqStr)
	}

	wc := cs.Downstream().(listeners.WrapConn)
	lanternDeviceID := req.Header.Get(common.DeviceIdHeader)
	if lanternDeviceID == "" {
		// Old lantern versions and possible cracks do not include the device
		// ID. Just throttle them.
		f.instrument.Throttle(req.Context(), true, "no-device-id")
		wc.ControlMessage("throttle", alwaysThrottle)
		return next(cs, req)
	}
	if lanternDeviceID == "~~~~~~" {
		// This is checkfallbacks, don't throttle it
		f.instrument.Throttle(req.Context(), false, "checkfallbacks")
		return next(cs, req)
	}

	// Even if a device hasn't hit its data cap, we always throttle to a default throttle rate to
	// keep bandwidth hogs from using too much bandwidth. Note - this does not apply to pro proxies
	// which don't use the devicefilter at all.
	throttleDefault := func(message string) {
		if defaultThrottleRate <= 0 {
			f.instrument.Throttle(req.Context(), false, message)
		}
		limiter := f.rateLimiterForDevice(lanternDeviceID, defaultThrottleRate, defaultThrottleRate)
		if log.IsTraceEnabled() {
			log.Tracef("Throttling connection to %v per second by default",
				humanize.Bytes(uint64(defaultThrottleRate)))
		}
		f.instrument.Throttle(req.Context(), true, "default")
		wc.ControlMessage("throttle", limiter)
	}

	// Some domains are excluded from being throttled and don't count towards the
	// bandwidth cap.
	if domains.ConfigForRequest(req).Unthrottled {
		throttleDefault("domain-excluded")
		return next(cs, req)
	}

	// Check usage from cache only - no eager fetching
	u := usage.Get(lanternDeviceID)
	if u == nil {
		// No usage data available yet, allow the request
		f.instrument.Throttle(req.Context(), false, "no-usage-data")
		return next(cs, req)
	}

	settings, err := f.datacapClient.GetDatacapUsage(req.Context(), lanternDeviceID)
	if err != nil {
		log.Errorf("failed to get datacap usage for device %s: %v", lanternDeviceID, err)
		f.instrument.Throttle(req.Context(), false, "datacap-error")
		//allow the request to proceed if we fail to get datacap usage
		settings = &TrackDatacapResponse{
			Allowed: true,
		}
	}

	measuredCtx := map[string]interface{}{
		"throttled": false,
	}

	var capOn bool

	// To turn the datacap off we simply set the threshold to 0 or below
	if settings.Allowed {
		log.Tracef("Got datacap settings: %v", settings)
		capOn = settings.CapLimit > 0

		measuredCtx["datacap_settings"] = settings
		if capOn {
			measuredCtx["datacap_threshold"] = settings.CapLimit
			measuredCtx["datacap_usage"] = u.Bytes
		}
	}

	if capOn && u.Bytes > settings.CapLimit {
		f.instrument.Throttle(req.Context(), true, "over-datacap")
		measuredCtx["throttled"] = true
		limiter := f.rateLimiterForDevice(lanternDeviceID, defaultThrottleRate, defaultThrottleRate)
		if log.IsTraceEnabled() {
			log.Tracef("Throttling connection from device %s to %v per second", lanternDeviceID,
				humanize.Bytes(uint64(defaultThrottleRate)))
		}
		f.instrument.Throttle(req.Context(), true, "datacap")
		wc.ControlMessage("throttle", limiter)
		measuredCtx["throttled"] = true
	} else {
		throttleDefault("")
	}

	wc.ControlMessage("measured", measuredCtx)

	resp, nextCtx, err := next(cs, req)
	if resp == nil || err != nil {
		return resp, nextCtx, err
	}
	if !capOn || !f.sendXBQHeader {
		return resp, nextCtx, err
	}
	if resp.Header == nil {
		resp.Header = make(http.Header, 1)
	}
	uMiB := u.Bytes / (1024 * 1024)
	xbq := fmt.Sprintf("%d/%d/%d", uMiB, settings.CapLimit/(1024*1024), int64(u.AsOf.Sub(epoch).Seconds()))
	xbqv2 := fmt.Sprintf("%s/%d", xbq, u.TTLSeconds)
	resp.Header.Set(common.XBQHeader, xbq)     // for backward compatibility with older clients
	resp.Header.Set(common.XBQHeaderv2, xbqv2) // for new clients that support different bandwidth cap expirations
	f.instrument.XBQHeaderSent(req.Context())
	return resp, nextCtx, err
}

func (f *deviceFilter) rateLimiterForDevice(deviceID string, rateLimitRead, rateLimitWrite int64) *listeners.RateLimiter {
	f.limitersByDeviceMx.Lock()
	defer f.limitersByDeviceMx.Unlock()

	limiter := f.limitersByDevice[deviceID]
	if limiter == nil || limiter.GetRateRead() != rateLimitRead || limiter.GetRateWrite() != rateLimitWrite {
		limiter = listeners.NewRateLimiter(rateLimitRead, rateLimitWrite)
		f.limitersByDevice[deviceID] = limiter
	}
	return limiter
}
