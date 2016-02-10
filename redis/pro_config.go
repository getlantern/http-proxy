package redis

import (
	"gopkg.in/redis.v3"
)

type proConfig struct {
	redisClient *redis.Client
}

func NewProConfig(redisAddr string) (*proConfig, error) {
	rc, err := getClient(redisAddr)
	if err != nil {
		return nil, err
	}
	return &proConfig{rc}, nil
}
