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
	redisClient *redis.Client
}

func NewMeasuredReporter(redisOpts *Options) (measured.Reporter, error) {
	rc, err := getClient(redisOpts)
	if err != nil {
		return nil, err
	}

	return &measuredReporter{
		redisClient: rc,
	}, nil
}

func (rp *measuredReporter) ReportTraffic(tt map[string]*measured.TrafficTracker) error {
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

		clientKey := "_client:" + key
		bytesIn, err := rp.redisClient.HIncrBy(clientKey, "bytesIn", int64(t.TotalIn)).Result()
		if err != nil {
			return err
		}
		bytesOut, err := rp.redisClient.HIncrBy(clientKey, "bytesOut", int64(t.TotalOut)).Result()
		if err != nil {
			return err
		}
		rp.redisClient.ExpireAt(clientKey, endOfThisMonth).Err()
		if err != nil {
			return err
		}

		usage.Set(key, uint64(bytesIn+bytesOut), now)
	}
	return nil
}
