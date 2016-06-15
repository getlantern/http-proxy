package redis

import (
	"errors"
	"time"

	"github.com/getlantern/http-proxy-lantern/usage"
	"gopkg.in/redis.v3"
)

func ForceRetrieveDeviceUsage(rc *redis.Client, device string) error {
	vals, err := rc.HMGet("_client:"+device, "bytesIn", "bytesOut").Result()
	if err != nil {
		return err
	}
	if len(vals) != 2 {
		return errors.New("Received unexpected values from Redis")
	}

	bytesIn := vals[0].(uint64)
	bytesOut := vals[1].(uint64)
	usage.Set(device, bytesIn+bytesOut, time.Now())
	return nil
}
