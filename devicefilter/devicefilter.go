package devicefilter

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"time"

	"github.com/dustin/go-humanize"

	"github.com/getlantern/golog"
	"github.com/getlantern/proxy/filters"

	"github.com/getlantern/http-proxy/listeners"

	"github.com/getlantern/http-proxy-lantern/blacklist"
	"github.com/getlantern/http-proxy-lantern/common"
	lanternlisteners "github.com/getlantern/http-proxy-lantern/listeners"
	"github.com/getlantern/http-proxy-lantern/redis"
	"github.com/getlantern/http-proxy-lantern/throttle"
	"github.com/getlantern/http-proxy-lantern/usage"
)

var (
	log = golog.LoggerFor("devicefilter")

	epoch = time.Date(2016, 1, 1, 0, 0, 0, 0, time.UTC)

	alwaysThrottle = lanternlisteners.NewRateLimiter(10)
)

// deviceFilterPre does the device-based filtering
type deviceFilterPre struct {
	deviceFetcher    *redis.DeviceFetcher
	throttleConfig   throttle.Config
	fasttrackDomains *common.FasttrackDomains
	sendXBQHeader    bool
}

// deviceFilterPost cleans up
type deviceFilterPost struct {
	bl *blacklist.Blacklist
}

// NewPre creates a filter which throttling all connections from a device if its data usage threshold is reached.
// * df is used to fetch device data usage across all proxies from a central Redis.
// * throttleConfig is to determine the threshold and throttle rate. They can
// be fixed values or fetched from Redis periodically.
// * If fasttrackDomains is given, it skips throttling for the fasttrackDomains, if any.
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

func NewPre(df *redis.DeviceFetcher, throttleConfig throttle.Config, fasttrackDomains *common.FasttrackDomains, sendXBQHeader bool) filters.Filter {
	if throttleConfig != nil {
		log.Debug("Throttling enabled")
	}

	return &deviceFilterPre{
		deviceFetcher:    df,
		throttleConfig:   throttleConfig,
		fasttrackDomains: fasttrackDomains,
		sendXBQHeader:    sendXBQHeader,
	}
}

func (f *deviceFilterPre) Apply(ctx filters.Context, req *http.Request, next filters.Next) (*http.Response, filters.Context, error) {
	if log.IsTraceEnabled() {
		reqStr, _ := httputil.DumpRequest(req, true)
		log.Tracef("DeviceFilter Middleware received request:\n%s", reqStr)
	}

	// If we're requesting a whitelisted domain, don't count it towards the
	// bandwidth cap.
	if f.fasttrackDomains.Whitelisted(req) {
		return next(ctx, req)
	}

	// Attached the uid to connection to report stats to redis correctly
	// "conn" in context is previously attached in server.go
	wc := ctx.DownstreamConn().(listeners.WrapConn)
	lanternDeviceID := req.Header.Get(common.DeviceIdHeader)

	if lanternDeviceID == "" {
		// Old lantern versions and possible cracks do not include the device
		// ID. Just throttle them.
		wc.ControlMessage("throttle", alwaysThrottle)
		return next(ctx, req)
	}
	if lanternDeviceID == "~~~~~~" {
		// This is checkfallbacks, don't throttle it
		return next(ctx, req)
	}

	if f.throttleConfig == nil {
		return next(ctx, req)
	}

	// Throttling enabled
	u := usage.Get(lanternDeviceID)
	if u == nil {
		// Eagerly request device ID data from Redis and store it in usage
		f.deviceFetcher.RequestNewDeviceUsage(lanternDeviceID)
		return next(ctx, req)
	}
	threshold, rate, ok := f.throttleConfig.ThresholdAndRateFor(lanternDeviceID, u.CountryCode)
  	// To turn the data cap off in redis we simply set the threshold to 0 or below. This
	// will also turn off the cap in the UI on desktop and in newer versions on mobile.
	capOn := threshold > 0
	if capOn && ok && u.Bytes > threshold {
		// per connection limiter
		limiter := lanternlisteners.NewRateLimiter(rate)
		log.Debugf("Throttling connection from device %s to %v per second", lanternDeviceID,
			humanize.Bytes(uint64(rate)))
		wc.ControlMessage("throttle", limiter)
	}

	resp, nextCtx, err := next(ctx, req)
	if resp == nil || err != nil {
		return resp, nextCtx, err
	}
	if !ok || !f.sendXBQHeader || !capOn {
		return resp, nextCtx, err
	}
	if resp.Header == nil {
		resp.Header = make(http.Header, 1)
	}
	uMiB := u.Bytes / (1024 * 1024)
	xbq := fmt.Sprintf("%d/%d/%d", uMiB, threshold/(1024*1024), int64(u.AsOf.Sub(epoch).Seconds()))
	resp.Header.Set(common.XBQHeader, xbq)
	return resp, nextCtx, err
}

func NewPost(bl *blacklist.Blacklist) filters.Filter {
	return &deviceFilterPost{
		bl: bl,
	}
}

func (f *deviceFilterPost) Apply(ctx filters.Context, req *http.Request, next filters.Next) (*http.Response, filters.Context, error) {
	// For privacy, delete the DeviceId header before passing it along
	req.Header.Del(common.DeviceIdHeader)
	ip, _, _ := net.SplitHostPort(req.RemoteAddr)
	f.bl.Succeed(ip)
	return next(ctx, req)
}
