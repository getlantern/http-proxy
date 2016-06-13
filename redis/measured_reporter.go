package redis

import (
	"net"
	"time"

	"gopkg.in/redis.v3"

	"github.com/getlantern/measured"

	"github.com/getlantern/http-proxy-lantern/usage"
)

var (
	keysExpiration = time.Hour * 24 * 31
)

type measuredReporter struct {
	redisClient   *redis.Client
	redisInterval *time.Ticker
}

func NewMeasuredReporter(redisOpts *Options, redisInterval time.Duration) (measured.Reporter, error) {
	rc, err := getClient(redisOpts)
	if err != nil {
		return nil, err
	}

	log.Debug("Will report traffic")

	return &measuredReporter{
		redisClient:   rc,
		redisInterval: time.NewTicker(redisInterval),
	}, nil
}

func (rp *measuredReporter) ReportTraffic(tt map[string]*measured.TrafficTracker) error {
	select {
	case <-rp.redisInterval.C:
		log.Debug("Reporting traffic to Redis")
		now := time.Now()
		nextMonth := now.Month() + 1
		nextYear := now.Year()
		if nextMonth > time.December {
			nextMonth = time.January
			nextYear++
		}
		beginningOfNextMonth := time.Date(nextYear, nextMonth, 1, 0, 0, 0, 0, now.Location())
		endOfThisMonth := beginningOfNextMonth.Add(-1 * time.Nanosecond)
		for key, t := range tt {
			// Don't report IDs in the form ip:port, so no connection that isn't
			// associated to a request that passes through devicefilter gets reported
			if _, _, err := net.SplitHostPort(key); err == nil {
				continue
			}

			multi := rp.redisClient.Multi()
			var bytesInOp *redis.IntCmd
			var bytesOutOp *redis.IntCmd
			_, merr := multi.Exec(func() error {
				clientKey := "_client:" + key
				// If any of these commands fails, the error will be immediately returned by Exec,
				// so we shouldn't be checking them here. Also, reifying the values should be done
				// after the Exec is done and we've checked for errors running it.
				bytesInOp = multi.HIncrBy(clientKey, "bytesIn", int64(t.TotalIn))
				bytesOutOp = multi.HIncrBy(clientKey, "bytesOut", int64(t.TotalOut))
				multi.ExpireAt(clientKey, endOfThisMonth)
				return nil
			})
			multi.Close()
			if merr != nil {
				return merr
			}

			bytesIn := bytesInOp.Val()
			bytesOut := bytesOutOp.Val()
			usage.Set(key, uint64(bytesIn+bytesOut), now)
		}
	default:
		for key, t := range tt {
			usage.Set(key, uint64(t.TotalIn+t.TotalOut), time.Now())
		}
	}
	return nil
}
