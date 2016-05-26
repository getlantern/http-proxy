package profilter

import (
	"errors"

	"github.com/getlantern/http-proxy-lantern/redis"
)

type proConfig struct {
	serverId    string
	redisConfig *redis.ProConfig
	userTokens  redis.UserTokens
	proFilter   *LanternProFilter
}

func NewRedisProConfig(redisOpts *redis.Options, serverId string, proFilter *LanternProFilter) (*proConfig, error) {
	redisConfig, err := redis.NewProConfig(redisOpts, serverId)
	if err != nil {
		return nil, err
	}
	return &proConfig{
		serverId:    serverId,
		redisConfig: redisConfig,
		userTokens:  make(redis.UserTokens),
		proFilter:   proFilter,
	}, nil
}

func (c *proConfig) processUserSetMessage(msg []string) error {
	// Should receive USER-SET,<USER>,<TOKEN>
	if len(msg) != 3 {
		return errors.New("Malformed SET message")
	}
	user := msg[1]
	token := msg[2]
	c.userTokens[user] = token
	return nil
}

func (c *proConfig) processUserRemoveMessage(msg []string) error {
	// Should receive USER-REMOVE,<USER>
	if len(msg) != 2 {
		return errors.New("Malformed REMOVE message")
	}
	user := msg[1]
	if _, ok := c.userTokens[user]; !ok {
		return errors.New("User in REMOVE message was not assigned to server")
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

func (c *proConfig) IsPro() (bool, error) {
	return c.redisConfig.IsPro()
}

func (c *proConfig) Run(initAsPro bool) error {
	initialize := func() (err error) {
		c.redisConfig.EmptyMessageQueue()

		c.proFilter.Enable()
		if c.userTokens, err = c.redisConfig.GetUsersAndTokens(); err != nil {
			return
		}
		c.proFilter.SetTokens(c.getAllTokens()...)
		return
	}

	if initAsPro {
		if err := initialize(); err != nil {
			return err
		}
	}

	go func() {
		for {
			msg, err := c.redisConfig.GetNextMessage()
			if err != nil {
				log.Debugf("Error reading message from Redis: %v", err)
				continue
			}
			switch msg[0] {
			case "TURN-PRO":
				initialize()
				log.Debug("Proxy now is Pro-only. Tokens updated.")
			case "TURN-FREE":
				c.proFilter.Disable()
				c.proFilter.ClearTokens()
				log.Debug("Proxy now is Free-only")
			case "USER-SET":
				// Add or update a user
				if err := c.processUserSetMessage(msg); err != nil {
					log.Errorf("Error setting user/token: %v", err)
				} else {
					// We need to update all tokens to avoid leaking old ones,
					// in case of token update
					c.proFilter.SetTokens(c.getAllTokens()...)
					log.Tracef("User added/updated. Complete set of users: %v", c.userTokens)
				}
			case "USER-REMOVE":
				if err := c.processUserRemoveMessage(msg); err != nil {
					log.Errorf("Error retrieving removed users/token: %v", err)
				} else {
					c.proFilter.SetTokens(c.getAllTokens()...)
					log.Tracef("Removed user. Current set: %v", c.userTokens)
				}
			default:
				log.Error("Unknown message type")
			}
		}
	}()
	return nil
}
