package proxy

import (
	"context"
	"net"
	"strings"
	"sync/atomic"
	"time"

	rclient "github.com/go-redis/redis/v8"

	"github.com/getlantern/geo"
	"github.com/getlantern/http-proxy/listeners"
	"github.com/getlantern/measured"

	"github.com/getlantern/http-proxy-lantern/v2/instrument"
	"github.com/getlantern/http-proxy-lantern/v2/redis"
	"github.com/getlantern/http-proxy-lantern/v2/throttle"
)

var (
	measuredReportingInterval = 1 * time.Minute
)

type reportingConfig struct {
	enabled bool
	wrapper func(ls net.Listener) net.Listener
}

func newReportingConfig(countryLookup geo.CountryLookup, rc *rclient.Client, instrument instrument.Instrument, throttleConfig throttle.Config) *reportingConfig {
	var numReporters int64
	go func() {
		for {
			time.Sleep(5 * time.Second)
			log.Debugf("numProxiedBytesReporters: %d", atomic.LoadInt64(&numReporters))
		}
	}()

	proxiedBytesReporter := func(ctx map[string]interface{}, stats *measured.Stats, deltaStats *measured.Stats, final bool) {
		atomic.AddInt64(&numReporters, 1)
		defer atomic.AddInt64(&numReporters, -1)

		if deltaStats.SentTotal == 0 && deltaStats.RecvTotal == 0 {
			// nothing to report
			return
		}
		// Note - sometimes we're missing the platform and version
		platform := ""
		_platform := ctx["app_platform"]
		if _platform != nil {
			platform = _platform.(string)
		}
		version := ""
		_version := ctx["app_version"]
		if _version != nil {
			version = _version.(string)
		}
		app := ""
		_app := ctx["app"]
		if _app != nil {
			app = strings.ToLower(_app.(string))
		}
		locale := ""
		_locale := ctx["locale"]
		if _locale != nil {
			locale = strings.ToLower(_locale.(string))
		}
		var client_ip net.IP
		_client_ip := ctx["client_ip"]
		if _client_ip != nil {
			client_ip = net.ParseIP(_client_ip.(string))
		}
		_deviceID := ctx["deviceid"]
		deviceID := ""
		if _deviceID != nil {
			deviceID = _deviceID.(string)
		}
		dataCapCohort := ""
		throttleSettings, hasThrottleSettings := ctx["throttle_settings"]
		if hasThrottleSettings {
			dataCapCohort = throttleSettings.(*throttle.Settings).Label
		}
		_originHost := ctx["origin_host"]
		originHost := ""
		if _originHost != nil {
			originHost = _originHost.(string)
		}
		instrument.ProxiedBytes(context.Background(), deltaStats.SentTotal, deltaStats.RecvTotal, platform, version, app, locale, dataCapCohort, client_ip, deviceID, originHost)
	}

	var reporter listeners.MeasuredReportFN
	if throttleConfig == nil {
		log.Debug("No throttling configured, don't bother reporting bandwidth usage to Redis")
		reporter = func(ctx map[string]interface{}, stats *measured.Stats, deltaStats *measured.Stats,
			final bool) {
			// noop
		}
	} else if rc != nil {
		reporter = redis.NewMeasuredReporter(countryLookup, rc, measuredReportingInterval, throttleConfig)
	}
	reporter = combineReporter(reporter, proxiedBytesReporter)
	wrapper := func(ls net.Listener) net.Listener {
		return listeners.NewMeasuredListener(ls, measuredReportingInterval, reporter)
	}
	return &reportingConfig{true, wrapper}
}

func combineReporter(reporters ...listeners.MeasuredReportFN) listeners.MeasuredReportFN {
	return func(ctx map[string]interface{}, stats *measured.Stats, deltaStats *measured.Stats, final bool) {
		for _, r := range reporters {
			r(ctx, stats, deltaStats, final)
		}
	}
}

func neverWrap(ls net.Listener) net.Listener {
	return ls
}
