package proxy

import (
	"context"
	"net"
	"strings"
	"time"

	rclient "github.com/go-redis/redis/v8"

	"github.com/getlantern/geo"
	"github.com/getlantern/http-proxy-lantern/v2/common"
	"github.com/getlantern/http-proxy-lantern/v2/listeners"
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
	proxiedBytesReporter := func(ctx map[string]interface{}, stats *measured.Stats, deltaStats *measured.Stats, final bool) {
		if deltaStats.SentTotal == 0 && deltaStats.RecvTotal == 0 {
			// nothing to report
			return
		}
		// Note - sometimes we're missing the platform and version
		platform := fromContext(ctx, common.Platform)
		platformVersion := fromContext(ctx, common.PlatformVersion)
		appVersion := fromContext(ctx, common.AppVersion)
		libraryVersion := fromContext(ctx, common.LibraryVersion)
		app := lowerFromContext(ctx, common.App)
		locale := lowerFromContext(ctx, common.Locale)
		deviceID := fromContext(ctx, common.DeviceID)
		originHost := fromContext(ctx, common.OriginHost)
		probingError := fromContext(ctx, common.ProbingError)
		arch := fromContext(ctx, common.KernelArch)

		//used for unbounded only
		unboundedTeamId := fromContext(ctx, common.UnboundedTeamId)
		log.Debugf("reporting stats to otel: unbounded_team_id:%s, bytes:%v", unboundedTeamId, stats.RecvTotal)

		var client_ip net.IP
		_client_ip := ctx[common.ClientIP]
		if _client_ip != nil {
			client_ip = net.ParseIP(_client_ip.(string))
		}

		dataCapCohort := ""
		throttleSettings, hasThrottleSettings := ctx[common.ThrottleSettings]
		if hasThrottleSettings {
			dataCapCohort = throttleSettings.(*throttle.Settings).Label
		}

		ctxWithTeamId := context.WithValue(context.Background(), common.UnboundedTeamId, unboundedTeamId)
		instrument.ProxiedBytes(ctxWithTeamId, deltaStats.SentTotal, deltaStats.RecvTotal, platform, platformVersion, libraryVersion, appVersion, app, locale, dataCapCohort, probingError, client_ip, deviceID, originHost, arch)
	}

	var reporter listeners.MeasuredReportFN
	if throttleConfig == nil {
		log.Debug("No throttling configured, don't bother reporting bandwidth usage to Redis")
		reporter = func(ctx map[string]interface{}, stats *measured.Stats, deltaStats *measured.Stats, final bool) {
			// noop
		}
	} else if rc != nil {
		reporter = redis.NewMeasuredReporter(countryLookup, rc, measuredReportingInterval, throttleConfig)
	}
	reporter = combineReporter(reporter, proxiedBytesReporter)

	wrapper := func(ls net.Listener) net.Listener {
		customDuration := 1 * time.Second
		return listeners.NewMeasuredListener(ls, customDuration, reporter)
	}
	return &reportingConfig{true, wrapper}
}

func fromContext(ctx map[string]interface{}, key string) string {
	value := ctx[key]
	if value != nil {
		return value.(string)
	}
	return ""
}

func lowerFromContext(ctx map[string]interface{}, key string) string {
	return strings.ToLower(fromContext(ctx, key))
}

func combineReporter(reporters ...listeners.MeasuredReportFN) listeners.MeasuredReportFN {
	return func(ctx map[string]interface{}, stats *measured.Stats, deltaStats *measured.Stats, final bool) {
		for _, r := range reporters {
			r(ctx, stats, deltaStats, final)
		}
	}
}
