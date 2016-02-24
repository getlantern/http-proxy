package redis

import (
	"fmt"

	"gopkg.in/redis.v3"

	"github.com/getlantern/golog"
)

var (
	log = golog.LoggerFor("redis")
	rcs = make(map[string]*redis.Client)
)

func getClient(redisAddr string) (*redis.Client, error) {
	if rc, ok := rcs[redisAddr]; ok {
		return rc, nil
	} else {
		rc := redis.NewClient(&redis.Options{
			Addr: redisAddr,
		})
		_, err := rc.Ping().Result()
		if err != nil {
			return nil, fmt.Errorf("Unable to ping redis server: %s", err)
		}
		rcs[redisAddr] = rc
		return rc, nil
	}
}
