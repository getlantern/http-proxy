// Package throttle provides the ability to read throttling configurations from
// redis. Configurations are stored in redis as maps under the keys
// "_throttle:desktop" and "_throttle:mobile". The key/value pairs in each map
// are the 2-digit lowercase ISO-3166 country code plus a pipe-delimited
// threshold and rate, for example:
//
//   _throttle:mobile
//     ""   "524288000|10240"
//     "cn" "104857600|10240"
//
package throttle

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"gopkg.in/redis.v3"

	"github.com/dustin/go-humanize"
	"github.com/getlantern/golog"
)

const (
	desktopSuffix = "desktop"
	mobileSuffix  = "mobile"
)

var (
	log = golog.LoggerFor("flashlight.throttle")
)

// Config is a per-country throttling config
type Config interface {
	// ThresholdAndRateFor returns the threshold (bytes) and throttled rate (bytes
	// per second) for the given deviceID in the given countryCode. If no country
	// found, returns the values for the blank "" countryCode which is used as a
	// default.
	ThresholdAndRateFor(deviceID string, countryCode string) (int64, int64)
}

type thresholdAndRate [2]int64

// threshold is in bytes
func (tar thresholdAndRate) threshold() int64 {
	return tar[0]
}

// rate is in bytes per second
func (tar thresholdAndRate) rate() int64 {
	return tar[1]
}

type config struct {
	rc              *redis.Client
	refreshInterval time.Duration
	desktop         map[string]*thresholdAndRate
	mobile          map[string]*thresholdAndRate
	mx              sync.RWMutex
}

// NewRedisConfig returns a new Config that uses the given redis client to load
// its configuration information and reload that information every
// refreshInterval.
func NewRedisConfig(rc *redis.Client, refreshInterval time.Duration) (Config, error) {
	desktop, err := loadLimits(rc, desktopSuffix)
	if err != nil {
		return nil, err
	}
	mobile, err := loadLimits(rc, mobileSuffix)
	if err != nil {
		return nil, err
	}
	cfg := &config{
		rc:              rc,
		refreshInterval: refreshInterval,
		desktop:         desktop,
		mobile:          mobile,
	}
	go cfg.keepCurrent()
	return cfg, nil
}

func loadLimits(rc *redis.Client, suffix string) (map[string]*thresholdAndRate, error) {
	key := "_throttle:" + suffix
	_limits, err := rc.HGetAllMap(key).Result()
	if err != nil {
		return nil, fmt.Errorf("Unable to read %v from redis: %v", key, err)
	}
	limits := make(map[string]*thresholdAndRate, len(_limits))
	for country, limit := range _limits {
		parts := strings.Split(limit, "|")
		if len(parts) != 2 {
			log.Errorf("Invalid config in %v for country %v: %v", key, country, limit)
			continue
		}
		threshold, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			log.Errorf("Invalid threshold in %v for country %v: %v", key, country, parts[0])
			continue
		}
		rate, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			log.Errorf("Invalid rate in %v for country %v: %v", key, country, parts[1])
			continue
		}
		limits[country] = &thresholdAndRate{threshold, rate}
	}

	defaultTR, hasDefault := limits[""]
	if !hasDefault {
		return nil, fmt.Errorf(`No default ("") country configured in %v!`, key)
	}

	threshold, rate := defaultTR.threshold(), defaultTR.rate()
	log.Debugf("Throttling %v by default to %v per second after %v", suffix, humanize.Bytes(uint64(rate)), humanize.Bytes(uint64(threshold)))

	return limits, nil
}

func (cfg *config) keepCurrent() {
	log.Debugf("Refreshing every %v", cfg.refreshInterval)
	for {
		time.Sleep(cfg.refreshInterval)
		desktop, err := loadLimits(cfg.rc, desktopSuffix)
		if err != nil {
			log.Error(err)
			continue
		}
		mobile, err := loadLimits(cfg.rc, mobileSuffix)
		if err != nil {
			log.Error(err)
			continue
		}
		cfg.mx.Lock()
		cfg.desktop = desktop
		cfg.mobile = mobile
		cfg.mx.Unlock()
		log.Debug("Refreshed")
	}
}

func (cfg *config) ThresholdAndRateFor(deviceID string, countryCode string) (int64, int64) {
	isDesktop := len(deviceID) == 8
	var limits map[string]*thresholdAndRate
	cfg.mx.RLock()
	if isDesktop {
		limits = cfg.desktop
	} else {
		limits = cfg.mobile
	}
	cfg.mx.RUnlock()
	tr, found := limits[countryCode]
	if !found {
		tr = limits[countryCode]
	}
	return tr.threshold(), tr.rate()
}
