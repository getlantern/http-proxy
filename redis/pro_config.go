package redis

import (
	"errors"
	"fmt"
	"gopkg.in/redis.v3"
	"strings"

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

func (c *proConfig) getNextMessage() ([]string, error) {
	// This will block until there is a message in this list
	if r, err := c.redisClient.BLPop(0, "server-msg:"+c.serverId).Result(); err != nil {
		return nil, fmt.Errorf("Error retrieving message: %v", err)
	} else {
		// The returned result is [key, value]
		return strings.Split(r[1], ","), nil
	}
}

func (c *proConfig) retrieveUsersAndTokens() (err error) {
	errored := 0

	if users, err := c.redisClient.SMembers("server->users:" + c.serverId).Result(); err != nil {
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

func (c *proConfig) processUserAddMessage(msg []string) error {
	// Should receive USER-ADD,<USER>,<TOKEN>
	if len(msg) != 3 {
		return errors.New("Malformed ADD message")
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
	isPro, err := c.redisClient.Exists("server->users:" + c.serverId).Result()
	if err != nil {
		return false, err
	}
	return isPro, nil
}

func (c *proConfig) Run(initAsPro bool) error {
	initialize := func() error {
		if err := c.redisClient.Del("server-msg:" + c.serverId).Err(); err != nil {
			return err
		}

		c.proFilter.Enable()
		if err := c.retrieveUsersAndTokens(); err != nil {
			log.Errorf("Error retrieving assigned users/tokens: %v", err)
		} else {
			c.proFilter.UpdateTokens(c.getAllTokens())
		}
		return nil
	}

	// Currently, this is never reached.  This is here to support an eventual
	// and likely separation between Free proxies and Pro proxies in two server queues
	if initAsPro {
		if err := initialize(); err != nil {
			return err
		}
	}

	go func() {
		for {
			msg, err := c.getNextMessage()
			if err != nil {
				continue
			}
			switch msg[0] {
			case "TURN-PRO":
				initialize()
				log.Debug("Proxy now is Pro-only. Retrieved tokens.")
			case "TURN-FREE":
				c.proFilter.Disable()
				c.proFilter.ClearTokens()
				log.Debug("Proxy now is Free-only")
			case "USER-ADD":
				if err := c.processUserAddMessage(msg); err != nil {
					log.Errorf("Error retrieving added user/token: %v", err)
				} else {
					c.proFilter.UpdateTokens(c.getAllTokens())
					log.Tracef("Added user: %v", c.userTokens)
				}
			case "USER-REMOVE":
				if err := c.processUserRemoveMessage(msg); err != nil {
					log.Errorf("Error retrieving removed users/token: %v", err)
				} else {
					c.proFilter.UpdateTokens(c.getAllTokens())
					log.Tracef("Removed user: %v", c.userTokens)
				}
			default:
				log.Error("Unknown message type")
			}
		}
	}()
	return nil
}
