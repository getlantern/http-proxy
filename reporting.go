package proxy

import (
	"net"
	"time"

	"github.com/getlantern/http-proxy/listeners"
	"github.com/getlantern/measured"
	rclient "gopkg.in/redis.v5"

	"github.com/getlantern/http-proxy-lantern/geo"
	"github.com/getlantern/http-proxy-lantern/instrument"
	"github.com/getlantern/http-proxy-lantern/redis"
)

var (
	measuredReportingInterval = 1 * time.Minute

	noReport = &reportingConfig{false, neverWrap}
)

type reportingConfig struct {
	enabled bool
	wrapper func(ls net.Listener) net.Listener
}

func newReportingConfig(geolookup geo.Lookup, rc *rclient.Client, enabled bool, bordaReporter listeners.MeasuredReportFN, instrument instrument.Instrument) *reportingConfig {
	if !enabled || rc == nil {
		return noReport
	}
	proxiedBytesReporter := func(ctx map[string]interface{}, stats *measured.Stats, deltaStats *measured.Stats, final bool) {
		if deltaStats.SentTotal == 0 && deltaStats.RecvTotal == 0 {
			// nothing to report
			return
		}
		platform := ""
		_platform := ctx["app_platform"]
		if _platform != nil {
			platform = _platform.(string)
		}
		version := ""
		_version := ctx["app_version"]
		if _platform != nil {
			version = _version.(string)
		}
		// Note - sometimes we're missing the platform and version
		instrument.ProxiedBytes(deltaStats.SentTotal, deltaStats.RecvTotal, platform, version)
	}
	reporter := redis.NewMeasuredReporter(geolookup, rc, measuredReportingInterval)
	if bordaReporter != nil {
		reporter = combineReporter(reporter, bordaReporter, proxiedBytesReporter)
	} else {
		reporter = combineReporter(reporter, proxiedBytesReporter)
	}
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

func neverReport(ctx map[string]interface{}, stats *measured.Stats, deltaStats *measured.Stats, final bool) {
}

func neverWrap(ls net.Listener) net.Listener {
	return ls
}
