package devicefilter

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"time"

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
)

// deviceFilterPre does the device-based filtering
type deviceFilterPre struct {
	deviceFetcher    *redis.DeviceFetcher
	throttleConfig   throttle.Config
	fasttrackDomains *common.FasttrackDomains
}

// deviceFilterPost cleans up
type deviceFilterPost struct {
	bl *blacklist.Blacklist
}

func NewPre(df *redis.DeviceFetcher, throttleConfig throttle.Config, fasttrackDomains *common.FasttrackDomains) filters.Filter {
	if throttleConfig != nil {
		log.Debug("Throttling enabled")
	}

	return &deviceFilterPre{
		deviceFetcher:    df,
		throttleConfig:   throttleConfig,
		fasttrackDomains: fasttrackDomains,
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
		// Old lantern versions and possible cracks do not include the device ID. Just throttle them.
		wc.ControlMessage("throttle", lanternlisteners.ThrottleRate(10))
	} else {
		if f.throttleConfig != nil {
			resp, nextCtx, err := next(ctx, req)
			// Throttling enabled
			u := usage.Get(lanternDeviceID)
			if u == nil {
				// Eagerly request device ID data to Redis and store it in usage
				f.deviceFetcher.RequestNewDeviceUsage(lanternDeviceID)
				return resp, nextCtx, err
			}
			if resp == nil || err != nil {
				return resp, nextCtx, err
			}
			uMiB := u.Bytes / (1024 * 1024)
			// Encode usage information in a header. The header is expected to follow
			// this format:
			//
			// <used>/<allowed>/<asof>
			//
			// <used> is the string representation of a 64-bit unsigned integer
			// <allowed> is the string representation of a 64-bit unsigned integer
			// <asof> is the 64-bit signed integer representing seconds since a custom
			// epoch (00:00:00 01/01/2016 UTC).
			threshold, rate := f.throttleConfig.ThresholdAndRateFor(lanternDeviceID, u.CountryCode)
			if resp.Header == nil {
				resp.Header = make(http.Header, 1)
			}
			resp.Header.Set(common.XBQHeader, fmt.Sprintf("%d/%d/%d", uMiB, threshold/(1024*1024), int64(u.AsOf.Sub(epoch).Seconds())))
			if u.Bytes > threshold {
				wc.ControlMessage("throttle", lanternlisteners.ThrottleRate(rate))
			}
			return resp, nextCtx, err
		}
	}

	return next(ctx, req)
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
