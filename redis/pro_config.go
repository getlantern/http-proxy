package redis

import (
	"fmt"
	"strconv"
	"strings"

	"gopkg.in/redis.v3"
)

type UserTokens map[string]string

type ProConfig struct {
	redisClient *redis.Client
	serverId    string
}

func NewProConfig(redisOpts *Options, serverId string) (*ProConfig, error) {
	rc, err := getClient(redisOpts)
	if err != nil {
		return nil, err
	}
	return &ProConfig{
		redisClient: rc,
		serverId:    serverId,
	}, nil
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

	if users, err := c.redisClient.SMembers("server->users:" + c.serverId).Result(); err != nil {
		return userTokens, err
	} else {
		log.Tracef("Assigned users: %v", users)
		if tokens, err := c.redisClient.HMGet("user->token", users...).Result(); err != nil {
			return userTokens, err
		} else {
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
		}
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

func (c *ProConfig) RetrieveGlobalUserDevices() (userDevices map[uint64][]string) {
	userDevices = make(map[uint64][]string)
	if users, err := c.redisClient.SMembers("server->users:" + c.serverId).Result(); err != nil {
		log.Errorf("Error retrieving server users")
	} else {
		log.Tracef("Assigned users: %v", users)
		for _, u := range users {
			devices, err := c.redisClient.SMembers("user->devices:" + u).Result()
			if err != nil {
				log.Errorf("Error retrieving user devices for user %v", u)
			} else {
				userID, convErr := strconv.ParseUint(u, 10, 64)
				if convErr != nil {
					log.Errorf("Error parsing user ID")
				} else {
					for i, d := range devices {
						pos := strings.IndexByte(d, '|')
						if pos == -1 {
							devices[i] = d
						} else {
							devices[i] = d[:pos]
						}
					}
					userDevices[userID] = devices
				}
			}
		}
	}
	return
}
