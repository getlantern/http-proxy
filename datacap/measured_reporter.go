package datacap

import (
	"context"
	"math/rand"
	"net"
	"time"

	"github.com/getlantern/geo"
	"github.com/getlantern/golog"
	"github.com/getlantern/http-proxy-lantern/v2/common"
	"github.com/getlantern/http-proxy-lantern/v2/listeners"
	"github.com/getlantern/http-proxy-lantern/v2/usage"
	"github.com/getlantern/measured"
)

var (
	log = golog.LoggerFor("datacap")
)

type statsAndContext struct {
	ctx   map[string]interface{}
	stats *measured.Stats
}

func (sac *statsAndContext) add(other *statsAndContext) *statsAndContext {
	newStats := *other.stats
	if sac != nil {
		newStats.SentTotal += sac.stats.SentTotal
		newStats.RecvTotal += sac.stats.RecvTotal
	}
	return &statsAndContext{other.ctx, &newStats}
}

// NewMeasuredReporter creates a new reporter that sends usage data to the sidecar service
func NewMeasuredReporter(countryLookup geo.CountryLookup, sidecarClient DatacapSidecarClient, reportInterval time.Duration) listeners.MeasuredReportFN {
	// Provide some buffering so that we don't lose data while submitting to the sidecar
	statsCh := make(chan *statsAndContext, 10000)
	go reportPeriodically(countryLookup, sidecarClient, reportInterval, statsCh)

	return func(ctx map[string]interface{}, stats *measured.Stats, deltaStats *measured.Stats, final bool) {
		select {
		case statsCh <- &statsAndContext{ctx, deltaStats}:
			// submitted successfully
		default:
			// data lost, probably because sidecar submission is taking longer than expected
			log.Error("Failed to queue stats for sidecar submission - queue is full")
		}
	}
}

func reportPeriodically(countryLookup geo.CountryLookup, sidecarClient DatacapSidecarClient, reportInterval time.Duration, statsCh chan *statsAndContext) {
	// randomize the interval to evenly distribute traffic
	randomized := time.Duration(reportInterval.Nanoseconds()/2 + rand.Int63n(reportInterval.Nanoseconds()))
	log.Debugf("Will report data usage to sidecar every %v", randomized)
	ticker := time.NewTicker(randomized)
	statsByDeviceID := make(map[string]*statsAndContext)

	for {
		select {
		case sac := <-statsCh:
			_deviceID := sac.ctx[common.DeviceID]
			if _deviceID == nil {
				// ignore requests without device ID
				continue
			}
			deviceID := _deviceID.(string)
			statsByDeviceID[deviceID] = statsByDeviceID[deviceID].add(sac)

		case <-ticker.C:
			if log.IsTraceEnabled() {
				log.Tracef("Submitting %d stats to sidecar", len(statsByDeviceID))
			}

			// Submit stats to sidecar
			err := submitToSidecar(context.Background(), countryLookup, sidecarClient, statsByDeviceID)
			if err != nil {
				log.Errorf("Unable to submit stats to sidecar: %v", err)
			}

			// Reset stats
			statsByDeviceID = make(map[string]*statsAndContext)
		}
	}
}

func submitToSidecar(ctx context.Context, countryLookup geo.CountryLookup, client DatacapSidecarClient, statsByDeviceID map[string]*statsAndContext) error {
	now := time.Now()

	for deviceID, sac := range statsByDeviceID {
		stats := sac.stats

		// Extract client IP for country lookup
		_clientIP := sac.ctx[common.ClientIP]
		if _clientIP == nil {
			log.Error("Missing client_ip in context, this shouldn't happen. Ignoring.")
			continue
		}
		clientIP := _clientIP.(string)
		countryCode := countryLookup.CountryCode(net.ParseIP(clientIP))

		// Extract platform
		platform := "unknown"
		_platform := sac.ctx[common.Platform]
		if _platform != nil {
			platform = _platform.(string)
		}

		// Calculate total bytes used
		totalBytes := stats.RecvTotal + stats.SentTotal
		if totalBytes <= 0 {
			continue
		}

		// Call sidecar to track usage
		trackResp, err := client.TrackDatacapUsage(ctx, deviceID, int64(totalBytes), countryCode, platform)
		if err != nil {
			log.Errorf("Failed to track usage for device %s: %v", deviceID, err)
			continue
		}

		// Store the result in local usage cache
		// TTL will be provided by the sidecar, default to 1 day if not provided
		ttl := int64(24 * 60 * 60) // 1 day in seconds as default
		usage.Set(deviceID, countryCode, int64(trackResp.RemainingBytes), now, ttl)

		if !trackResp.Allowed {
			log.Debugf("Device %s has reached its data cap", deviceID)
		}
	}

	return nil
}
