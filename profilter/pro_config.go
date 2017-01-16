package profilter

import (
	"errors"

	"github.com/getlantern/http-proxy-lantern/redis"
	redislib "gopkg.in/redis.v3"
)

type proConfig struct {
	serverId    string
	redisConfig *redis.ProConfig
	userTokens  redis.UserTokens
	userDevices redis.UserDevices
	proFilter   *lanternProFilter
}

func NewRedisProConfig(rc *redislib.Client, serverId string, proFilter *lanternProFilter) *proConfig {
	return &proConfig{
		serverId:    serverId,
		redisConfig: redis.NewProConfig(rc, serverId),
		userTokens:  make(redis.UserTokens),
		userDevices: make(redis.UserDevices),
		proFilter:   proFilter,
	}
}

func (c *proConfig) processUserSetMessage(msg []string) error {
	// Should receive USER-SET,<USER>,<TOKEN>
	if len(msg) != 3 {
		return errors.New("Malformed SET message")
	}
	user := msg[1]
	token := msg[2]
	c.userTokens[user] = token

	// Get user devices too
	devs, err := c.redisConfig.GetUserDevices(user)
	if err != nil {
		return errors.New("Error retrieving user devices in UPDATE-DEVICES message")
	}
	c.userDevices[user] = devs

	return nil
}

func (c *proConfig) processUserRemoveMessage(msg []string) error {
	// Should receive USER-REMOVE,<USER>
	if len(msg) != 2 {
		return errors.New("Malformed REMOVE message")
	}

	user := msg[1]

	// Remove the user from the devices registry (don't care if the user has no
	// device registered)
	delete(c.userDevices, user)

	// Remove the user from the tokens registry (the main one used for
	// user->server assignation knowledge in proxies
	if _, ok := c.userTokens[user]; !ok {
		return errors.New("User in REMOVE message was not assigned to server")
	}

	delete(c.userTokens, user)
	return nil
}

func (c *proConfig) processUserUpdateDevicesMessage(msg []string) error {
	// Should receive USER-UPDATE-DEVICES,<USER>
	if len(msg) != 2 {
		return errors.New("Malformed UPDATE-DEVICES message")
	}
	user := msg[1]
	if _, ok := c.userTokens[user]; !ok {
		return errors.New("User in UPDATE-DEVICES message was not assigned to server")
	}

	devs, err := c.redisConfig.GetUserDevices(user)
	if err != nil {
		return errors.New("Error retrieving user devices in UPDATE-DEVICES message")
	}
	c.userDevices[user] = devs

	return nil
}

// updateAllDevices retrieves the devices from all proxy users
func (c *proConfig) updateAllDevices() {
	for user := range c.userTokens {
		devices, err := c.redisConfig.GetUserDevices(user)
		if err != nil {
			log.Debugf("Error retrieving devices for user %d: %v", user, err)
		} else {
			c.userDevices[user] = devices
		}
	}
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

func (c *proConfig) getAllDevices() []string {
	// 3 is the number of max devices per user
	devices := make([]string, 0, len(c.userDevices)*3)
	for _, udevs := range c.userDevices {
		for _, d := range udevs {
			devices = append(devices, d)
		}
	}
	return devices
}

func (c *proConfig) IsPro() (bool, error) {
	return c.redisConfig.IsPro()
}

func (c *proConfig) Run(initAsPro bool) error {
	initialize := func() (err error) {
		if c.userTokens, err = c.redisConfig.GetUsersAndTokens(); err != nil {
			return
		}

		// Initialize only if there are users assigned to this server
		if len(c.userTokens) > 0 {
			c.proFilter.Enable()
		} else {
			log.Debugf("The proxy has no assigned users: Free-only proxy.")
			return
		}

		tks := c.getAllTokens()
		c.proFilter.SetTokens(tks...)

		c.updateAllDevices()
		devices := c.getAllDevices()
		c.proFilter.SetDevices(devices...)

		log.Debugf("The proxy has assigned users: Pro-only proxy.")
		log.Debugf("Initializing with the following Pro tokens: %v", tks)
		log.Debugf("Initializing with the following allowed devices: %v", devices)
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
			case "USER-SET":
				c.redisConfig.EmptyMessageQueue()
				// If this is the first user of the proxy, initialization will be required
				if len(c.userTokens) == 0 {
					initialize()
				}
				// Add or update an user
				if err := c.processUserSetMessage(msg); err != nil {
					log.Errorf("Error setting user/token: %v", err)
				} else {
					// We need to update all tokens to avoid leaking old ones,
					// in case of token update
					c.proFilter.SetTokens(c.getAllTokens()...)
					c.proFilter.SetDevices(c.getAllDevices()...)
					log.Tracef("User added/updated. Complete set of users: %v", c.userTokens)
				}
			case "USER-REMOVE":
				if err := c.processUserRemoveMessage(msg); err != nil {
					log.Errorf("Error retrieving removed users/token: %v", err)
				} else {
					c.proFilter.SetTokens(c.getAllTokens()...)
					c.proFilter.SetDevices(c.getAllDevices()...)
					log.Tracef("Removed user. Current set: %v", c.userTokens)
				}
			case "USER-UPDATE-DEVICES":
				if err := c.processUserUpdateDevicesMessage(msg); err != nil {
					log.Errorf("Error updating user devices: %v", err)
				} else {
					c.proFilter.SetDevices(c.getAllDevices()...)
				}
			case "TURN-PRO":
				initialize()
				log.Debug("Proxy now is Pro-only. Tokens updated.")
			case "TURN-FREE":
				c.proFilter.Disable()
				c.proFilter.ClearTokens()
				log.Debug("Proxy now is Free-only")
			default:
				log.Error("Unknown message type")
			}
		}
	}()
	return nil
}
