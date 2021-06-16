package redis

import (
	"crypto/tls"
	"fmt"
	"regexp"

	"github.com/go-redis/redis/v8"
)

func NewClient(redisURL string) (*redis.Client, error) {
	pass, host, err := parseRedisURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %v", err)
	}

	return redis.NewClient(&redis.Options{
		Addr:     host,
		Password: pass,
		TLSConfig: &tls.Config{
			ClientSessionCache: tls.NewLRUClientSessionCache(20),
		},
	}), nil
}

func parseRedisURL(redisURL string) (password string, host string, err error) {
	redisURLRegExp := regexp.MustCompile(`^rediss://.*?:(.*?)@([\d\.(:\d*)?,]*)$`)
	matches := redisURLRegExp.FindStringSubmatch(redisURL)
	if len(matches) < 3 {
		return "", "", fmt.Errorf("%s should match %s", redisURL, redisURLRegExp.String())
	}
	return matches[1], matches[2], nil
}
