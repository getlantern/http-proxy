package redis

import (
	"fmt"
	"gopkg.in/redis.v3"

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

func (c *proConfig) retrieveUsersAndTokens() (err error) {
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

func (c *proConfig) processUserAddMessage() error {
	user, err := c.redisClient.LPop("server-msg:" + c.serverId).Result()
	if err != nil {
		return fmt.Errorf("malformed add user message (user not present) - %v", err)
	}
	token, err := c.redisClient.LPop("server-msg:" + c.serverId).Result()
	if err != nil {
		return fmt.Errorf("malformed add user message (token not present) - %v", err)
	}
	c.userTokens[user] = token
	return nil
}

func (c *proConfig) processUserRemoveMessage() error {
	user, err := c.redisClient.LPop("server-msg:" + c.serverId).Result()
	if err != nil {
		return fmt.Errorf("malformed add user message (user not present)")
	}
	delete(c.userTokens, user)
	return nil
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

func (c *proConfig) Run(initAsPro bool) {
	initialize := func() {
		c.proFilter.Enable()
		if err := c.retrieveUsersAndTokens(); err != nil {
			log.Errorf("Error retrieving assigned users/tokens: %v", err)
		} else {
			c.proFilter.UpdateTokens(c.getAllTokens())
		}
	}

	if initAsPro {
		initialize()
	}

	go func() {
		for {
			switch c.getNextMessage() {
			case "turn-pro":
				initialize()
				log.Debug("Proxy now is Pro-only. Retrieved tokens.")
			case "turn-free":
				c.proFilter.Disable()
				c.proFilter.ClearTokens()
				log.Debug("Proxy now is Free-only")
			case "user-add":
				if err := c.processUserAddMessage(); err != nil {
					log.Errorf("Error retrieving added user/token: %v", err)
				} else {
					c.proFilter.UpdateTokens(c.getAllTokens())
				}
				log.Debugf("Adding user: %v", c.userTokens)
			case "user-remove":
				if err := c.processUserRemoveMessage(); err != nil {
					log.Errorf("Error retrieving removed users/token: %v", err)
				} else {
					c.proFilter.UpdateTokens(c.getAllTokens())
				}
				log.Debugf("Removing user: %v", c.userTokens)
			default:
			}
		}
	}()
}
