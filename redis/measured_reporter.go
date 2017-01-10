package redis

import (
	"time"

	"gopkg.in/redis.v3"

	"github.com/getlantern/http-proxy-lantern/usage"
	"github.com/getlantern/http-proxy/listeners"
	"github.com/getlantern/measured"
)

var (
	keysExpiration = time.Hour * 24 * 31
)

type statsAndContext struct {
	ctx   map[string]interface{}
	stats *measured.Stats
}

func NewMeasuredReporter(rc *redis.Client, reportInterval time.Duration) listeners.MeasuredReportFN {
	// Provide some buffering so that we don't lose data while submitting to Redis
	statsCh := make(chan *statsAndContext, 10000)
	log.Debug("Will report traffic")
	return func(ctx map[string]interface{}, stats *measured.Stats, deltaStats *measured.Stats, final bool) {
		select {
		case statsCh <- &statsAndContext{ctx, deltaStats}:
			// submitted successfull
		default:
			// data lost, probably because Redis submission is taking longer than expected
		}
	}
}

func reportPeriodically(rc *redis.Client, reportInterval time.Duration, statsCh chan (*statsAndContext)) {
	log.Debug("Reporting traffic")
	ticker := time.NewTicker(reportInterval)
	statsByDeviceID := make(map[string]*measured.Stats)

	for {
		select {
		case sac := <-statsCh:
			_deviceID := sac.ctx["deviceid"]
			if _deviceID == nil {
				// ignore
				continue
			}
			deviceID := _deviceID.(string)
			existing := statsByDeviceID[deviceID]
			if existing == nil {
				existing = sac.stats
			} else {
				existing.SentTotal += sac.stats.SentTotal
				existing.RecvTotal += sac.stats.RecvTotal
			}
		case <-ticker.C:
			err := submit(rc, statsByDeviceID)
			if err != nil {
				log.Errorf("Unable to submit stats: %v", err)
			}
			// Reset stats
			statsByDeviceID = make(map[string]*measured.Stats)
		}
	}
}

func submit(rc *redis.Client, statsByDeviceID map[string]*measured.Stats) error {
	now := time.Now()
	nextMonth := now.Month() + 1
	nextYear := now.Year()
	if nextMonth > time.December {
		nextMonth = time.January
		nextYear++
	}
	beginningOfNextMonth := time.Date(nextYear, nextMonth, 1, 0, 0, 0, 0, now.Location())
	endOfThisMonth := beginningOfNextMonth.Add(-1 * time.Nanosecond)
	for deviceID, stats := range statsByDeviceID {
		multi := rc.Multi()
		var bytesInOp *redis.IntCmd
		var bytesOutOp *redis.IntCmd
		_, merr := multi.Exec(func() error {
			clientKey := "_client:" + deviceID
			// If any of these commands fails, the error will be immediately returned by Exec,
			// so we shouldn't be checking them here. Also, reifying the values should be done
			// after the Exec is done and we've checked for errors running it.
			bytesInOp = multi.HIncrBy(clientKey, "bytesIn", int64(stats.RecvTotal))
			bytesOutOp = multi.HIncrBy(clientKey, "bytesOut", int64(stats.SentTotal))
			multi.ExpireAt(clientKey, endOfThisMonth)
			return nil
		})
		multi.Close()
		if merr != nil {
			return merr
		}

		bytesIn := bytesInOp.Val()
		bytesOut := bytesOutOp.Val()
		usage.Set(deviceID, uint64(bytesIn+bytesOut), now)
	}
	return nil
}
