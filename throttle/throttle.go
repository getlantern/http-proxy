// Package throttle provides the ability to read throttling configurations from
// redis. Configurations are stored in redis as maps under the keys
// "_throttle:desktop" and "_throttle:mobile". The key/value pairs in each map
// are the 2-digit lowercase ISO-3166 country code plus a pipe-delimited
// threshold and rate, for example:
//
//   _throttle:mobile
//     "__"   "524288000|10240"
//     "cn"   "104857600|10240"
//
package throttle

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"gopkg.in/redis.v5"

	"github.com/dustin/go-humanize"
	"github.com/getlantern/golog"
)

const (
	// DesktopSuffix is the suffix for desktop config entries in Redis
	DesktopSuffix = "desktop"
	// MobileSuffix is the suffix for mobile config entries in Redis
	MobileSuffix = "mobile"
	// DefaultCountryCode is the field for default limits in Redis
	DefaultCountryCode = "__"

	DefaultRefreshInterval = 5 * time.Minute
)

var (
	log = golog.LoggerFor("flashlight.throttle")
)

// Config is a per-country throttling config
type Config interface {
	// ThresholdAndRateFor returns the threshold (bytes) and throttled rate (bytes
	// per second) for the given deviceID in the given countryCode. If no country
	// found, returns the values for the blank "__" countryCode which is used as a
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

func parseThresholdAndRate(limit string) (*thresholdAndRate, error) {
	parts := strings.Split(limit, "|")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid config: %v", limit)
	}
	threshold, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid threshold: %v", parts[0])
	}
	rate, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid rate : %v", parts[1])
	}
	return &thresholdAndRate{threshold, rate}, nil
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
func NewRedisConfig(rc *redis.Client, refreshInterval time.Duration, forceThreshold int64, forceRate int64) (Config, error) {
	if forceThreshold > 0 && forceRate > 0 {
		log.Debugf("Forcing throttling threshold and rate to %d : %d", forceThreshold, forceRate)
		desktop := make(map[string]*thresholdAndRate)
		mobile := make(map[string]*thresholdAndRate)
		desktop[DefaultCountryCode] = &thresholdAndRate{forceThreshold, forceRate}
		mobile[DefaultCountryCode] = &thresholdAndRate{forceThreshold, forceRate}
		cfg := &config{
			rc:              rc,
			refreshInterval: refreshInterval,
			desktop:         desktop,
			mobile:          mobile,
		}
		return cfg, nil
	}

	desktop, err := loadLimits(rc, DesktopSuffix)
	if err != nil {
		return nil, err
	}
	mobile, err := loadLimits(rc, MobileSuffix)
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
	_limits, err := rc.HGetAll(key).Result()
	if err != nil {
		return nil, fmt.Errorf("Unable to read %v from redis: %v", key, err)
	}
	limits := make(map[string]*thresholdAndRate, len(_limits))
	for country, limit := range _limits {
		tar, err := parseThresholdAndRate(limit)
		if err != nil {
			log.Errorf("For %v in country %v %v", key, country, err)
			continue
		}
		limits[country] = tar
	}

	defaultTR, hasDefault := limits[DefaultCountryCode]
	if !hasDefault {
		return nil, fmt.Errorf(`No default "__" country configured in %v!`, key)
	}

	threshold, rate := defaultTR.threshold(), defaultTR.rate()
	log.Debugf("Throttling %v by default to %v per second after %v", suffix, humanize.Bytes(uint64(rate)), humanize.Bytes(uint64(threshold)))

	return limits, nil
}

func (cfg *config) keepCurrent() {
	if cfg.refreshInterval <= 0 {
		log.Debugf("Defaulting refresh interval to %v", DefaultRefreshInterval)
		cfg.refreshInterval = DefaultRefreshInterval
	}

	log.Debugf("Refreshing every %v", cfg.refreshInterval)
	for {
		time.Sleep(cfg.refreshInterval)
		desktop, err := loadLimits(cfg.rc, DesktopSuffix)
		if err != nil {
			log.Error(err)
			continue
		}
		mobile, err := loadLimits(cfg.rc, MobileSuffix)
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
		tr = limits[DefaultCountryCode]
	}
	return tr.threshold(), tr.rate()
}
