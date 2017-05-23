package throttle

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"gopkg.in/redis.v3"

	"github.com/getlantern/golog"
)

var (
	log = golog.LoggerFor("flashlight.throttle")
)

// Config is a per-country throttling config
type Config interface {
	// ThresholdAndRateFor returns the threshold (bytes) and throttled rate (bytes
	// per second) for the given countryCode. If none found, returns the values
	// for the blank "" countryCode which is used as a default.
	ThresholdAndRateFor(countryCode string) (int64, int64)
}

type thresholdAndRate [2]int64

func (tar thresholdAndRate) threshold() int64 {
	return tar[0]
}

func (tar thresholdAndRate) rate() int64 {
	return tar[1]
}

type config struct {
	rc              *redis.Client
	refreshInterval time.Duration
	countries       map[string]thresholdAndRate
	mx              sync.RWMutex
}

// NewRedisConfig returns a new Config that uses the given redis client to load
// its configuration information and reload that information every
// refreshInterval.
func NewRedisConfig(rc *redis.Client, refreshInterval time.Duration) (Config, error) {
	countries, err := loadCountries(rc)
	if err != nil {
		return nil, err
	}
	cfg := &config{
		rc:              rc,
		refreshInterval: refreshInterval,
		countries:       countries,
	}
	go cfg.keepCurrent()
	return cfg, nil
}

func loadCountries(rc *redis.Client) (map[string]thresholdAndRate, error) {
	_countries, err := rc.HGetAllMap("_throttleConfig").Result()
	if err != nil {
		return nil, fmt.Errorf("Unable to read _throttleConfig from redis: %v", err)
	}
	countries := make(map[string]thresholdAndRate, len(_countries))
	for country, tstring := range _countries {
		parts := strings.Split(tstring, "|")
		if len(parts) != 2 {
			log.Errorf("Invalid config for country %v: %v", country, tstring)
			continue
		}
		threshold, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			log.Errorf("Invalid threshold for country %v: %v", country, parts[0])
			continue
		}
		rate, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			log.Errorf("Invalid rate for country %v: %v", country, parts[1])
			continue
		}
		countries[country] = thresholdAndRate{threshold, rate}
	}

	_, hasDefault := countries[""]
	if !hasDefault {
		return nil, fmt.Errorf(`No default ("") country configured!`)
	}

	return countries, nil
}

func (cfg *config) keepCurrent() {
	for {
		time.Sleep(cfg.refreshInterval)
		countries, err := loadCountries(cfg.rc)
		if err != nil {
			log.Error(err)
		} else {
			cfg.mx.Lock()
			cfg.countries = countries
			cfg.mx.Unlock()
			log.Debug("Refreshed")
		}
	}
}

func (cfg *config) ThresholdAndRateFor(countryCode string) (int64, int64) {
	cfg.mx.RLock()
	tr, found := cfg.countries[countryCode]
	if !found {
		tr = cfg.countries[countryCode]
	}
	cfg.mx.RUnlock()
	return tr.threshold(), tr.rate()
}
