package redis

import (
	"crypto/tls"

	"github.com/go-redis/redis/v8"
)

// Creates a new redis client with the specified redis URL to use, in the form:
// rediss://:password@host
func NewClient(redisURL string) (*redis.Client, error) {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		panic(err)
	}

	return redis.NewClient(&redis.Options{
		Addr:     opt.Addr,
		Password: opt.Password,
		TLSConfig: &tls.Config{
			ClientSessionCache: tls.NewLRUClientSessionCache(20),
		},
	}), nil
}
