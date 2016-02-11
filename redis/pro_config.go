package redis

import (
	"fmt"
	"gopkg.in/redis.v3"
	"time"

	"github.com/getlantern/http-proxy-lantern/profilter"
)

type proConfig struct {
	redisClient *redis.Client
	serverId    string
	proFilter   *profilter.LanternProFilter

	userTokens map[string]string
}

func NewProConfig(redisAddr string, serverId string, proFilter *profilter.LanternProFilter) (*proConfig, error) {
	rc, err := getClient(redisAddr)
	if err != nil {
		return nil, err
	}
	return &proConfig{
		redisClient: rc,
		serverId:    serverId,
		proFilter:   proFilter,
		userTokens:  make(map[string]string),
	}, nil
}

func (c *proConfig) getNextMessage() string {
	// This will block until there is a message in this list
	if r, err := c.redisClient.BLPop(0, "server-msg:"+c.serverId).Result(); err != nil {
		return "error"
	} else {
		// The returned result is [key, value]
		return r[1]
	}
}

func (c *proConfig) updateUsersAndTokens() (err error) {
	errored := 0

	if users, err := c.redisClient.LRange("server->users:"+c.serverId, 0, -1).Result(); err != nil {
		return err
	} else {
		log.Tracef("Assigned users: %v", users)
		if tokens, err := c.redisClient.HMGet("user->token", users...).Result(); err != nil {
			return err
		} else {
			log.Tracef("User tokens: %v", tokens)
			c.userTokens = make(map[string]string)
			i := 0
			for _, u := range users {
				if tk, ok := tokens[i].(string); ok {
					// Tokens are returned in order by HMGET
					c.userTokens[u] = tk
				} else {
					errored++
				}
				i++
			}
		}
	}
	if errored != 0 {
		return fmt.Errorf("critical! %d user(s) without token", errored)
	}
	return
}

func (c *proConfig) getAllTokens() []string {
	tokens := make([]string, len(c.userTokens))
	i := 0
	for _, v := range c.userTokens {
		tokens[i] = v
		i++
	}
	return tokens
}

func (c *proConfig) Run() {
	go func() {
		for {
			switch c.getNextMessage() {
			case "turn-pro":
				log.Debug("Proxy now is Pro-only. Retrieving tokens.")
				c.proFilter.Enable()
				if err := c.updateUsersAndTokens(); err != nil {
					log.Errorf("Error retrieving assigned users/tokens: %v", err)
				}
				tokens := c.getAllTokens()
				c.proFilter.UpdateTokens(tokens)
			case "turn-free":
				log.Debug("Proxy now is Free-only")
				c.proFilter.Disable()
				c.proFilter.ClearTokens()
			case "user-add":
				log.Debug("Adding user:")
				//c.proFilter.UpdateTokens(tokens)
			case "user-remove":
				log.Debug("Removing user:")
				//c.proFilter.UpdateTokens(tokens)
			case "error":
				log.Debug("Error retrieving messages from central Redis: waiting 30 seconds to retry")
				// TODO: After a few tries, ping Redis and/or reconnect
				time.Sleep(30 * time.Second)
			default:
			}
		}
	}()
}
