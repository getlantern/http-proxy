package redis

import (
	"fmt"
	"strings"

	"gopkg.in/redis.v3"
)

type UserTokens map[string]string

type ProConfig struct {
	redisClient *redis.Client
	serverId    string
}

func NewProConfig(rc *redis.Client, serverId string) *ProConfig {
	return &ProConfig{
		redisClient: rc,
		serverId:    serverId,
	}
}

func (c *ProConfig) EmptyMessageQueue() error {
	return c.redisClient.Del("server-msg:" + c.serverId).Err()
}

func (c *ProConfig) GetNextMessage() ([]string, error) {
	// This will block until there is a message in this list
	if r, err := c.redisClient.BLPop(0, "server-msg:"+c.serverId).Result(); err != nil {
		return nil, fmt.Errorf("Error retrieving message: %v", err)
	} else {
		// The returned result is [key, value]
		return strings.Split(r[1], ","), nil
	}
}

func (c *ProConfig) GetUsersAndTokens() (UserTokens, error) {
	userTokens := make(UserTokens)
	errored := 0

	users, err := c.redisClient.SMembers("server->users:" + c.serverId).Result()
	if err != nil {
		return userTokens, err
	}
	log.Debugf("Assigned users: %v", users)

	if len(users) == 0 {
		return userTokens, nil
	}

	tokens, err := c.redisClient.HMGet("user->token", users...).Result()
	if err != nil {
		return userTokens, err
	}
	log.Tracef("User tokens: %v", tokens)

	i := 0
	for _, u := range users {
		if tk, ok := tokens[i].(string); ok {
			// Tokens are returned in order by HMGET
			userTokens[u] = tk
		} else {
			errored++
		}
		i++
	}

	if errored != 0 {
		return userTokens, fmt.Errorf("critical! %d user(s) without token", errored)
	}
	return userTokens, nil
}

func (c *ProConfig) IsPro() (bool, error) {
	isPro, err := c.redisClient.Exists("server->users:" + c.serverId).Result()
	if err != nil {
		return false, err
	}
	return isPro, nil
}
