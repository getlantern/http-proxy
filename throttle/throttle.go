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
	"encoding/json"
	"strings"
	"sync"
	"time"

	"github.com/spaolacci/murmur3"
	"gopkg.in/redis.v5"

	"github.com/getlantern/golog"
)

const (
	DefaultRefreshInterval = 5 * time.Minute
)

var (
	log = golog.LoggerFor("flashlight.throttle")
)

type CapInterval string

const (
	Daily   = "daily"
	Weekly  = "weekly"
	Monthly = "monthly"
)

type ThrottleSettings struct {
	DeviceFloor float64
	DeviceCeil  float64
	// Threshold at which we start throttling (in bytes)
	Threshold int64
	// Rate to which to throttle (in bytes per second)
	Rate int64
	// How frequently the usage cap resets
	CapResets CapInterval
}

// Config is a per-country throttling config
type Config interface {
	// SettingsFor returns the throttling settings for the given deviceID in the given
	// countryCode on the given platform (windows, darwin, linux, android or ios). At the each level
	// (country and platform) this should fall back to default values if a specific value isn't provided.
	// Time zone is used to identify clients that are new enough to support anything other than monthly TTL
	SettingsFor(deviceID string, countryCode string, platform string, timeZone string) (settings *ThrottleSettings, ok bool)
}

// NewForcedConfig returns a new Config that uses the forced threshold, rate and TTL
func NewForcedConfig(threshold int64, rate int64, capResets CapInterval) Config {
	return &forcedConfig{
		ThrottleSettings: ThrottleSettings{
			Threshold: threshold,
			Rate:      rate,
			CapResets: capResets,
		},
	}
}

type forcedConfig struct {
	ThrottleSettings
}

func (cfg *forcedConfig) SettingsFor(deviceID string, countryCode string, platform string, timeZone string) (settings *ThrottleSettings, ok bool) {
	return &cfg.ThrottleSettings, true
}

// SavedThrottleSettings organizes slices of ThrottleSettingsWithConstraints by
// country -> platform
type SavedThrottleSettings map[string]map[string][]*ThrottleSettings

func decodeSavedThrottleSettings(encoded []byte) (settings SavedThrottleSettings, err error) {
	settings = make(SavedThrottleSettings)
	err = json.Unmarshal(encoded, &settings)
	return
}

type redisConfig struct {
	rc              *redis.Client
	refreshInterval time.Duration
	savedSettings   SavedThrottleSettings
	mx              sync.RWMutex
}

// NewRedisConfig returns a new Config that uses the given redis client to load
// its configuration information and reload that information every
// refreshInterval.
func NewRedisConfig(rc *redis.Client, refreshInterval time.Duration) Config {
	cfg := &redisConfig{
		rc:              rc,
		refreshInterval: refreshInterval,
	}
	cfg.refreshSettings()
	go cfg.keepCurrent()
	return cfg
}

func (cfg *redisConfig) keepCurrent() {
	if cfg.refreshInterval <= 0 {
		log.Debugf("Defaulting refresh interval to %v", DefaultRefreshInterval)
		cfg.refreshInterval = DefaultRefreshInterval
	}

	log.Debugf("Refreshing every %v", cfg.refreshInterval)
	for {
		time.Sleep(cfg.refreshInterval)
		cfg.refreshSettings()
	}
}

func (cfg *redisConfig) refreshSettings() {
	encoded, err := cfg.rc.Get("_throttle").Bytes()
	if err != nil {
		log.Errorf("Unable to throttle settings from redis: %v", err)
		return
	}
	settings, err := decodeSavedThrottleSettings(encoded)
	if err != nil {
		log.Errorf("Unable to decode throttle settings: %v", err)
		return
	}

	log.Debugf("Loaded throttle config: %v", string(encoded))

	cfg.mx.Lock()
	cfg.savedSettings = settings
	cfg.mx.Unlock()
}

func (cfg *redisConfig) SettingsFor(deviceID string, countryCode string, platform string, timeZone string) (settings *ThrottleSettings, ok bool) {
	cfg.mx.RLock()
	savedSettings := cfg.savedSettings
	cfg.mx.RUnlock()

	platformSettings, _ := savedSettings[strings.ToLower(countryCode)]
	if platformSettings == nil {
		log.Tracef("No settings found for country %v, use default", countryCode)
		platformSettings, _ = savedSettings["default"]
		if platformSettings == nil {
			log.Trace("No settings for default country, not throttling")
			return nil, false
		}
	}

	constrainedSettings, _ := platformSettings[strings.ToLower(platform)]
	if len(constrainedSettings) == 0 {
		log.Tracef("No settings found for platform %v, use default", platform)
		constrainedSettings, _ = platformSettings["default"]
		if len(constrainedSettings) == 0 {
			log.Trace("No settings for default platform, not throttling")
			return nil, false
		}
	}

	needsMonthly := timeZone == ""
	hash := murmur3.New64()
	hash.Write([]byte(deviceID))
	hashOfDeviceID := hash.Sum64()
	const scale = 1000000 // do not change this, as it will result in users being segmented differently than they were before
	segment := float64((hashOfDeviceID % scale)) / float64(scale)
	for _, candidateSettings := range constrainedSettings {
		if !needsMonthly || candidateSettings.CapResets == Monthly {
			if candidateSettings.DeviceFloor <= segment && (candidateSettings.DeviceCeil > segment || (candidateSettings.DeviceCeil == 1 && segment == 1)) {
				return candidateSettings, true
			}
		}
	}

	if needsMonthly {
		log.Tracef("No setting for segment %v, using first monthly in list", segment)
		for _, candidateSettings := range constrainedSettings {
			if candidateSettings.CapResets == Monthly {
				return candidateSettings, true
			}
		}
		log.Trace("No monthly cap available, don't throttle")
		return nil, false
	}

	log.Tracef("No setting for segment %v, using first in list", segment)
	return constrainedSettings[0], true
}
