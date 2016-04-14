package redis

import (
	"fmt"
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

func NewMeasuredReporter(redisAddr string) (measured.Reporter, error) {
	rc, err := getClient(redisAddr)
	if err != nil {
		return nil, err
	}
	return &measuredReporter{rc}, nil
}

func (rp *measuredReporter) ReportError(s map[*measured.Error]int) error {
	return nil
}
func (rp *measuredReporter) ReportLatency(s []*measured.LatencyTracker) error {
	return nil
}
func (rp *measuredReporter) ReportTraffic(tt []*measured.TrafficTracker) error {
	for _, t := range tt {
		key := t.ID
		if key == "" {
			panic("empty key is not allowed")
		}
		tx := rp.redisClient.Multi()
		defer tx.Close()

		_, err := tx.Exec(func() error {
			clientKey := "_client:" + string(key)

			err := tx.HIncrBy(clientKey, "bytesIn", int64(t.LastIn)).Err()
			if err != nil {
				return err
			}
			err = tx.HIncrBy(clientKey, "bytesOut", int64(t.LastOut)).Err()
			if err != nil {
				return err
			}
			err = tx.Expire(clientKey, keysExpiration).Err()
			if err != nil {
				return err
			}

			// An auxiliary ordered set for aggregated bytesIn+bytesOut
			// Redis stores scores as float64
			err = tx.ZAdd("client->bytes",
				redis.Z{
					float64(t.TotalIn + t.TotalOut),
					key,
				}).Err()
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
