package redis

import (
	"fmt"
	"net"
	"time"

	"gopkg.in/redis.v3"

	"github.com/getlantern/measured"
)

var (
	keysExpiration time.Duration = time.Hour * 24 * 31
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

		tx := rp.redisClient.Multi()
		defer tx.Close()

		_, err := tx.Exec(func() error {
			clientKey := "_client:" + string(key)

			err := tx.HIncrBy(clientKey, "bytesIn", int64(t.TotalIn)).Err()
			if err != nil {
				return err
			}
			err = tx.HIncrBy(clientKey, "bytesOut", int64(t.TotalOut)).Err()
			if err != nil {
				return err
			}
			err = tx.ExpireAt(clientKey, endOfThisMonth).Err()
			if err != nil {
				return err
			}

			// Ordered set for aggregated bytesIn+bytesOut, bytesIn, bytesOut
			// Redis stores scores as float64
			err = tx.ZIncrBy("client->bytesIn", float64(t.TotalIn), key).Err()
			if err != nil {
				return err
			}
			err = tx.ZIncrBy("client->bytesOut", float64(t.TotalOut), key).Err()
			if err != nil {
				return err
			}
			err = tx.ZIncrBy("client->bytesInOut", float64(t.TotalIn+t.TotalOut), key).Err()
			if err != nil {
				return err
			}

			return nil
		})
		if err != nil {
			return fmt.Errorf("Error in MULTI command: %v\n", err)
		}
	}
	return nil
}
