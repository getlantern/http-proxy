package proxy

import (
	"net"
	"time"

	"github.com/getlantern/measured"
	rclient "gopkg.in/redis.v5"

	"github.com/getlantern/http-proxy-lantern/redis"
	"github.com/getlantern/http-proxy/listeners"
)

var (
	measuredReportingInterval = 1 * time.Minute

	noReport = &reportingConfig{false, neverWrap}
)

type reportingConfig struct {
	enabled bool
	wrapper func(ls net.Listener) net.Listener
}

func newReportingConfig(rc *rclient.Client, enabled bool, bordaReporter listeners.MeasuredReportFN) *reportingConfig {
	if !enabled || rc == nil {
		return noReport
	}
	reporter := redis.NewMeasuredReporter(rc, measuredReportingInterval)
	if bordaReporter != nil {
		reporter = combineReporter(reporter, bordaReporter)
	}
	wrapper := func(ls net.Listener) net.Listener {
		return listeners.NewMeasuredListener(ls, measuredReportingInterval, reporter)
	}
	return &reportingConfig{true, wrapper}
}

func combineReporter(r1 listeners.MeasuredReportFN, r2 listeners.MeasuredReportFN) listeners.MeasuredReportFN {
	return func(ctx map[string]interface{}, stats *measured.Stats, deltaStats *measured.Stats, final bool) {
		r1(ctx, stats, deltaStats, final)
		r2(ctx, stats, deltaStats, final)
	}
}

func neverReport(ctx map[string]interface{}, stats *measured.Stats, deltaStats *measured.Stats, final bool) {
}

func neverWrap(ls net.Listener) net.Listener {
	return ls
}
