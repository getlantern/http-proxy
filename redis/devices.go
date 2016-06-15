package redis

import (
	"strconv"
	"time"

	"github.com/getlantern/http-proxy-lantern/usage"
	"gopkg.in/redis.v3"
)

func ForceRetrieveDeviceUsage(rc *redis.Client, device string) error {
	vals, err := rc.HMGet("_client:"+device, "bytesIn", "bytesOut").Result()
	if err != nil {
		return err
	} else if vals[0] == nil || vals[1] == nil {
		// No entry found or partially stored, nothing to be done
		return nil
	}

	bytesIn, err := strconv.ParseUint(vals[0].(string), 10, 64)
	if err != nil {
		return err
	}
	bytesOut, err := strconv.ParseUint(vals[1].(string), 10, 64)
	if err != nil {
		return err
	}

	usage.Set(device, bytesIn+bytesOut, time.Now())
	return nil
}
