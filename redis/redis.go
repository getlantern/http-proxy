package redis

import (
	"fmt"
	"net/url"

	"gopkg.in/redis.v3"

	"github.com/getlantern/golog"
)

var (
	log = golog.LoggerFor("redis")
	rcs = make(map[string]*redis.Client)
)

func getClient(redisUrl string) (*redis.Client, error) {
	u, err := url.Parse(redisUrl)
	if err != nil {
		return nil, fmt.Errorf("Unable to parse Redis address: %s", err)
	}

	if u.Host == "" {
		return nil, fmt.Errorf("Please provide a Redis URL of the form 'redis://[user:pass@host:port]'")
	}

	if rc, ok := rcs[u.Host]; ok {
		return rc, nil
	} else {
		opt := redis.Options{Addr: u.Host}
		if u.User != nil {
			redisPass, _ := u.User.Password()
			opt.Password = redisPass
		}

		rc := redis.NewClient(&opt)
		_, err := rc.Ping().Result()
		if err != nil {
			return nil, fmt.Errorf("Unable to ping redis server: %s", err)
		}
		rcs[u.Host] = rc
		return rc, nil
	}
}
