package devicefilter

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"sync"
	"time"

	"github.com/dustin/go-humanize"

	"github.com/getlantern/golog"
	"github.com/getlantern/proxy/v3/filters"

	"github.com/getlantern/http-proxy-lantern/v2/listeners"

	"github.com/getlantern/http-proxy-lantern/v2/blacklist"
	"github.com/getlantern/http-proxy-lantern/v2/common"
	"github.com/getlantern/http-proxy-lantern/v2/datacap"
	"github.com/getlantern/http-proxy-lantern/v2/domains"
	"github.com/getlantern/http-proxy-lantern/v2/instrument"
	"github.com/getlantern/http-proxy-lantern/v2/redis"
	"github.com/getlantern/http-proxy-lantern/v2/throttle"
	"github.com/getlantern/http-proxy-lantern/v2/usage"
)

var (
	log = golog.LoggerFor("devicefilter")

	epoch = time.Date(2016, 1, 1, 0, 0, 0, 0, time.UTC)

	alwaysThrottle = listeners.NewRateLimiter(10, 10) // this is basically unusably slow, only used for malicious or really old/broken clients
)

// DefaultThrottleRate is the ceiling every non-pro device is held to even
// before it reaches its data cap, so that no one device monopolizes a proxy.
const DefaultThrottleRate = int64(5000 * 1024 / 8) // 5 Mbps

// checkfallbacksDeviceID is the sentinel device ID checkfallbacks sends.
const checkfallbacksDeviceID = "~~~~~~"

// throttleSentinelDevice applies the policy for the two sentinel device-ID
// values, shared by both accounting paths so they cannot drift: no device ID
// (old lantern versions and possible cracks — throttled to a near-unusable
// rate) and the checkfallbacks marker (never throttled). It reports whether
// the request was one of the two.
func throttleSentinelDevice(inst instrument.Instrument, wc listeners.WrapConn, req *http.Request, deviceID string) bool {
	switch deviceID {
	case "":
		inst.Throttle(req.Context(), true, "no-device-id")
		wc.ControlMessage("throttle", alwaysThrottle)
		return true
	case checkfallbacksDeviceID:
		inst.Throttle(req.Context(), false, "checkfallbacks")
		return true
	}
	return false
}

// setXBQHeaders attaches the XBQ/XBQv2 usage headers flashlight's bandwidth
// package renders in the client UI. This is the single definition of that wire
// format for both accounting paths — a device must see identical headers no
// matter which path its proxy was provisioned with.
//
// XBQ is <used MiB>/<allowed MiB>/<seconds since epoch (2016-01-01 UTC)>;
// XBQv2 appends /<seconds until the cap resets>. XBQ is kept for backward
// compatibility with older clients.
func setXBQHeaders(resp *http.Response, usedBytes, capBytes int64, asOf time.Time, ttlSeconds int64) {
	if resp.Header == nil {
		resp.Header = make(http.Header, 1)
	}
	xbq := fmt.Sprintf("%d/%d/%d", usedBytes/(1024*1024), capBytes/(1024*1024), int64(asOf.Sub(epoch).Seconds()))
	resp.Header.Set(common.XBQHeader, xbq)
	resp.Header.Set(common.XBQHeaderv2, fmt.Sprintf("%s/%d", xbq, ttlSeconds))
}

// deviceFilterPre does the device-based filtering, with usage fetched from the
// reporting Redis. Its datacap-sidecar counterpart is datacapFilterPre.
type deviceFilterPre struct {
	deviceFetcher      *redis.DeviceFetcher
	throttleConfig     throttle.Config
	sendXBQHeader      bool
	instrument         instrument.Instrument
	limitersByDevice   map[string]*listeners.RateLimiter
	limitersByDeviceMx sync.Mutex
}

// deviceFilterPost cleans up
type deviceFilterPost struct {
	bl *blacklist.Blacklist
}

// NewPre creates a filter which throttling all connections from a device if its data usage threshold is reached.
// * df is used to fetch device data usage across all proxies from a central Redis.
// * throttleConfig is to determine the threshold and throttle rate. They can
// be fixed values or fetched from Redis periodically.
// * If sendXBQHeader is true, it attaches a common.XBQHeader to inform the
// clients the usage information before this request is made. The header is
// expected to follow this format:
//
// <used>/<allowed>/<asof>
//
// <used> is the string representation of a 64-bit unsigned integer
// <allowed> is the string representation of a 64-bit unsigned integer
// <asof> is the 64-bit signed integer representing seconds since a custom
// epoch (00:00:00 01/01/2016 UTC).
func NewPre(df *redis.DeviceFetcher, throttleConfig throttle.Config, sendXBQHeader bool, instrument instrument.Instrument) filters.Filter {
	if throttleConfig != nil {
		log.Debug("Throttling enabled")
	}

	return &deviceFilterPre{
		deviceFetcher:    df,
		throttleConfig:   throttleConfig,
		sendXBQHeader:    sendXBQHeader,
		instrument:       instrument,
		limitersByDevice: make(map[string]*listeners.RateLimiter, 0),
	}
}

func (f *deviceFilterPre) Apply(cs *filters.ConnectionState, req *http.Request, next filters.Next) (*http.Response, *filters.ConnectionState, error) {
	if log.IsTraceEnabled() {
		reqStr, _ := httputil.DumpRequest(req, true)
		log.Tracef("DeviceFilter Middleware received request:\n%s", reqStr)
	}

	// Attached the uid to connection to report stats to redis correctly
	// "conn" in context is previously attached in server.go
	wc := cs.Downstream().(listeners.WrapConn)
	lanternDeviceID := req.Header.Get(common.DeviceIdHeader)

	// Even if a device hasn't hit its data cap, we always throttle to a default throttle rate to
	// keep bandwidth hogs from using too much bandwidth. Note - this does not apply to pro proxies
	// which don't use the devicefilter at all.
	throttleDefault := func(message string) {
		if DefaultThrottleRate <= 0 {
			f.instrument.Throttle(req.Context(), false, message)
		}
		limiter := f.rateLimiterForDevice(lanternDeviceID, DefaultThrottleRate, DefaultThrottleRate)
		if log.IsTraceEnabled() {
			log.Tracef("Throttling connection to %v per second by default",
				humanize.Bytes(uint64(DefaultThrottleRate)))
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

	if throttleSentinelDevice(f.instrument, wc, req, lanternDeviceID) {
		return next(cs, req)
	}

	if f.throttleConfig == nil {
		f.instrument.Throttle(req.Context(), false, "no-config")
		return next(cs, req)
	}

	// Throttling enabled
	u := usage.Get(lanternDeviceID)
	if u == nil {
		// Eagerly request device ID data from Redis and store it in usage
		f.deviceFetcher.RequestNewDeviceUsage(lanternDeviceID)
		throttleDefault("no-usage-data")
		return next(cs, req)
	}

	settings, capOn := f.throttleConfig.SettingsFor(lanternDeviceID, u.CountryCode, req.Header.Get(common.PlatformHeader), req.Header.Get(common.AppHeader), req.Header[common.SupportedDataCapsHeader])

	measuredCtx := map[string]interface{}{
		"throttled": false,
	}

	// To turn the data cap off in Redis we simply set the threshold to 0 or
	// below. This will also turn off the cap in the UI on desktop and in newer
	// versions on mobile.
	if capOn {
		log.Tracef("Got throttle settings: %v", settings)
		capOn = settings.Threshold > 0

		// Send throttle settings to measured as well
		measuredCtx["throttle_settings"] = settings
	}

	if capOn && u.Bytes > settings.Threshold {
		// per connection limiter
		// Note - when people hit the data cap, we only throttle writes back to the client, not reads.
		// This way, they can continue to upload videos or other bandwidth intensive content for sharing.
		limiter := f.rateLimiterForDevice(lanternDeviceID, DefaultThrottleRate, settings.Rate)
		if log.IsTraceEnabled() {
			log.Tracef("Throttling connection from device %s to %v per second", lanternDeviceID,
				humanize.Bytes(uint64(settings.Rate)))
		}
		f.instrument.Throttle(req.Context(), true, "datacap")
		wc.ControlMessage("throttle", limiter)
		measuredCtx["throttled"] = true
	} else {
		// default case is not throttling
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
	setXBQHeaders(resp, u.Bytes, settings.Threshold, u.AsOf, u.TTLSeconds)
	f.instrument.XBQHeaderSent(req.Context())
	return resp, nextCtx, err
}

func (f *deviceFilterPre) rateLimiterForDevice(deviceID string, rateLimitRead, rateLimitWrite int64) *listeners.RateLimiter {
	f.limitersByDeviceMx.Lock()
	defer f.limitersByDeviceMx.Unlock()

	limiter := f.limitersByDevice[deviceID]
	if limiter == nil || limiter.GetRateRead() != rateLimitRead || limiter.GetRateWrite() != rateLimitWrite {
		limiter = listeners.NewRateLimiter(rateLimitRead, rateLimitWrite)
		f.limitersByDevice[deviceID] = limiter
	}
	return limiter
}

// datacapFilterPre is deviceFilterPre's counterpart for proxies whose byte
// accounting runs through the local datacap sidecar. The throttle decision
// itself is made asynchronously by the tracker; the filter attaches the
// device's shared limiter and surfaces the tracker's latest view of the device
// to the client via the XBQ headers.
type datacapFilterPre struct {
	tracker       *datacap.Tracker
	sendXBQHeader bool
	instrument    instrument.Instrument
}

// NewDatacapPre creates the filter for proxies whose byte accounting runs
// through the local datacap sidecar. Unlike the Redis path, the limiter it
// attaches is shared across all of a device's connections and is re-rated by
// the tracker as reports come back, so crossing the cap slows down transfers
// that are already in flight rather than only the next one.
func NewDatacapPre(tracker *datacap.Tracker, sendXBQHeader bool, instrument instrument.Instrument) filters.Filter {
	return &datacapFilterPre{
		tracker:       tracker,
		sendXBQHeader: sendXBQHeader,
		instrument:    instrument,
	}
}

func (f *datacapFilterPre) Apply(cs *filters.ConnectionState, req *http.Request, next filters.Next) (*http.Response, *filters.ConnectionState, error) {
	if log.IsTraceEnabled() {
		reqStr, _ := httputil.DumpRequest(req, true)
		log.Tracef("DeviceFilter Middleware received request:\n%s", reqStr)
	}

	wc := cs.Downstream().(listeners.WrapConn)
	deviceID := req.Header.Get(common.DeviceIdHeader)

	// Some domains are excluded from being throttled. Their bytes still count
	// towards the cap (accounting is per connection, not per request), they are
	// just never held to the capped rate — hence a separate limiter that the
	// tracker never re-rates.
	//
	// This check deliberately precedes the sentinel-device guards, matching the
	// Redis path: a request to an excluded domain gets the default rate even
	// with a missing device ID. Moving the guards first would newly subject
	// old clients to alwaysThrottle on domains we have decided not to throttle.
	// Such requests share one limiter under the empty device ID, exactly as
	// rateLimiterForDevice("") does on the Redis path.
	if domains.ConfigForRequest(req).Unthrottled {
		f.instrument.Throttle(req.Context(), true, "default")
		wc.ControlMessage("throttle", f.tracker.Limiter(deviceID, true))
		return next(cs, req)
	}

	if throttleSentinelDevice(f.instrument, wc, req, deviceID) {
		return next(cs, req)
	}

	// The limiter is shared by every connection of this device and already
	// carries whatever rate the last sidecar response set, so attaching it is
	// the whole of enforcement here.
	limiter, u, haveUsage := f.tracker.LimiterAndUsage(deviceID)
	wc.ControlMessage("throttle", limiter)

	if !haveUsage {
		// The sidecar only learns of a device from its first usage report, so
		// there is nothing to report to the client yet. The limiter is at the
		// default rate until then.
		f.instrument.Throttle(req.Context(), true, "default")
		return next(cs, req)
	}

	reason := "default"
	if u.Throttled {
		reason = "datacap"
	}
	f.instrument.Throttle(req.Context(), true, reason)

	resp, nextCtx, err := next(cs, req)
	if resp == nil || err != nil {
		return resp, nextCtx, err
	}
	// A zero cap limit means this device's country/platform has no cap entry,
	// which the client renders as "no data cap" — same meaning as a
	// non-positive Redis threshold did.
	if u.CapLimit <= 0 || !f.sendXBQHeader {
		return resp, nextCtx, err
	}
	ttlSeconds := int64(0)
	if !u.Expiry.IsZero() {
		if remaining := time.Until(u.Expiry); remaining > 0 {
			ttlSeconds = int64(remaining.Seconds())
		}
	}
	setXBQHeaders(resp, u.BytesUsed, u.CapLimit, u.AsOf, ttlSeconds)
	f.instrument.XBQHeaderSent(req.Context())
	return resp, nextCtx, err
}

func NewPost(bl *blacklist.Blacklist) filters.Filter {
	return &deviceFilterPost{
		bl: bl,
	}
}

func (f *deviceFilterPost) Apply(cs *filters.ConnectionState, req *http.Request, next filters.Next) (*http.Response, *filters.ConnectionState, error) {
	// For privacy, delete the DeviceId header before passing it along
	req.Header.Del(common.DeviceIdHeader)
	ip, _, _ := net.SplitHostPort(req.RemoteAddr)
	f.bl.Succeed(ip)
	return next(cs, req)
}
