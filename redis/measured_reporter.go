package redis

import (
	"net"
	"time"

	"gopkg.in/redis.v3"

	"github.com/getlantern/http-proxy-lantern/devicefilter"
	"github.com/getlantern/measured"
)

var (
	keysExpiration time.Duration = time.Hour * 24 * 31
)

type measuredReporter struct {
	redisClient      *redis.Client
	registerDeviceAt int64
}

func NewMeasuredReporter(redisOpts *Options, registerDeviceAt int64) (measured.Reporter, error) {
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

		pipe := rp.redisClient.Pipeline()
		defer pipe.Close()

		clientKey := "_client:" + key
		bytesIn, err := pipe.HIncrBy(clientKey, "bytesIn", int64(t.TotalIn)).Result()
		if err != nil {
			return err
		}
		bytesOut, err := pipe.HIncrBy(clientKey, "bytesOut", int64(t.TotalOut)).Result()
		if err != nil {
			return err
		}
		pipe.ExpireAt(clientKey, endOfThisMonth).Err()
		if err != nil {
			return err
		}

		// In both cases, we first check existence, since it provides lower contention
		if bytesIn+bytesOut >= rp.registerDeviceAt {
			if !devicefilter.DeviceRegistryExists(key) {
				devicefilter.DeviceRegistryAdd(key)
			}
		} else {
			if devicefilter.DeviceRegistryExists(key) {
				devicefilter.DeviceRegistryRemove(key)
			}
		}
	}
	return nil
}
