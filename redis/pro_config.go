package redis

import (
	"time"

	"gopkg.in/redis.v3"

	"github.com/getlantern/http-proxy-lantern/profilter"
)

type proConfig struct {
	redisClient *redis.Client
	serverId    string
	proFilter   *profilter.LanternProFilter
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
	}, nil
}

func (c *proConfig) getNextMessage() string {
	// This will block until there is a message in this list
	if r, err := c.redisClient.BLPop(0, "server-msg:"+c.serverId).Result(); err != nil {
		// The returned result is [key, value]
		return r[1]
	} else {
		return "error"
	}
}

func (c *proConfig) Run() {
	go func() {
		for {
			switch c.getNextMessage() {
			case "turn-pro":
				log.Debug("Proxy now is Pro-only")
				c.proFilter.Enable()
			case "turn-free":
				log.Debug("Proxy now is Free-only")
				c.proFilter.Disable()
			case "error":
				log.Debug("Error retrieving messages from central Redis: waiting 30 seconds to retry")
				time.Sleep(30 * time.Second)
			default:
			}
		}
	}()
}
