package redis

import (
	"math/rand"
	"net"
	"strconv"
	"strings"
	"time"

	"gopkg.in/redis.v5"

	"github.com/getlantern/geo"
	"github.com/getlantern/golog"
	"github.com/getlantern/http-proxy-lantern/v2/throttle"
	"github.com/getlantern/http-proxy-lantern/v2/usage"
	"github.com/getlantern/http-proxy/listeners"
	"github.com/getlantern/measured"
)

const updateUsageScript = `
	local clientKey = KEYS[1]

	local bytesIn = redis.call("hincrby", clientKey, "bytesIn", ARGV[1])
	local bytesOut = redis.call("hincrby", clientKey, "bytesOut", ARGV[2])
	local countryCode = redis.call("hget", clientKey, "countryCode")
	if not countryCode or countryCode == "" then
		countryCode = ARGV[3]
		redis.call("hset", clientKey, "countryCode", countryCode)
		-- record the IP on which we based the countryCode for auditing
		redis.call("hset", clientKey, "clientIP", ARGV[4])
		redis.call("expireat", clientKey, ARGV[5])
	end

	local ttl = redis.call("ttl", clientKey)
	return {bytesIn, bytesOut, countryCode, ttl}
`

var (
	log = golog.LoggerFor("redis")
)

type statsAndContext struct {
	ctx   map[string]interface{}
	stats *measured.Stats
}

func NewMeasuredReporter(countryLookup geo.CountryLookup, rc *redis.Client, reportInterval time.Duration, throttleConfig throttle.Config) listeners.MeasuredReportFN {
	// Provide some buffering so that we don't lose data while submitting to Redis
	statsCh := make(chan *statsAndContext, 10000)
	go reportPeriodically(countryLookup, rc, reportInterval, throttleConfig, statsCh)
	return func(ctx map[string]interface{}, stats *measured.Stats, deltaStats *measured.Stats, final bool) {
		select {
		case statsCh <- &statsAndContext{ctx, deltaStats}:
			// submitted successfully
		default:
			// data lost, probably because Redis submission is taking longer than expected
		}
	}
}

func reportPeriodically(countryLookup geo.CountryLookup, rc *redis.Client, reportInterval time.Duration, throttleConfig throttle.Config, statsCh chan *statsAndContext) {
	// randomize the interval to evenly distribute traffic to reporting Redis.
	randomized := time.Duration(reportInterval.Nanoseconds()/2 + rand.Int63n(reportInterval.Nanoseconds()))
	log.Debugf("Will report data usage to Redis every %v", randomized)
	ticker := time.NewTicker(randomized)
	statsByDeviceID := make(map[string]*statsAndContext)
	var scriptSHA string
	for {
		select {
		case sac := <-statsCh:
			_deviceID := sac.ctx["deviceid"]
			if _deviceID == nil {
				// ignore
				continue
			}
			deviceID := _deviceID.(string)
			existing := statsByDeviceID[deviceID]
			if existing == nil {
				statsByDeviceID[deviceID] = sac
			} else {
				existing.stats.SentTotal += sac.stats.SentTotal
				existing.stats.RecvTotal += sac.stats.RecvTotal
			}
		case <-ticker.C:
			if log.IsTraceEnabled() {
				log.Tracef("Submitting %d stats", len(statsByDeviceID))
			}
			if scriptSHA == "" {
				var err error
				scriptSHA, err = rc.ScriptLoad(updateUsageScript).Result()
				if err != nil {
					log.Errorf("Unable to load script, skip submitting stats: %v", err)
					continue
				}
			}

			err := submit(countryLookup, rc, scriptSHA, statsByDeviceID, throttleConfig)
			if err != nil {
				log.Errorf("Unable to submit stats: %v", err)
			}
			// Reset stats
			statsByDeviceID = make(map[string]*statsAndContext)
		}
	}
}

func submit(countryLookup geo.CountryLookup, rc *redis.Client, scriptSHA string, statsByDeviceID map[string]*statsAndContext, throttleConfig throttle.Config) error {
	for deviceID, sac := range statsByDeviceID {
		now := time.Now()
		stats := sac.stats

		_clientIP := sac.ctx["client_ip"]
		if _clientIP == nil {
			log.Error("Missing client_ip in context, this shouldn't happen. Ignoring.")
			continue
		}
		clientIP := _clientIP.(string)
		countryCode := countryLookup.CountryCode(net.ParseIP(clientIP))
		var platform string
		_platform, ok := sac.ctx["app_platform"]
		if ok {
			platform = _platform.(string)
		}
		var timeZone string
		_timeZone, ok := sac.ctx["timeZone"]
		if ok {
			timeZone = _timeZone.(string)
		} else {
			timeZone = now.Location().String()
		}
		throttleSettings, ok := throttleConfig.SettingsFor(deviceID, countryCode, platform, timeZone)
		if !ok {
			log.Trace("No throttle config, don't bother tracking usage")
			continue
		}

		clientKey := "_client:" + deviceID
		_result, err := rc.EvalSha(scriptSHA, []string{clientKey},
			strconv.Itoa(stats.RecvTotal),
			strconv.Itoa(stats.SentTotal),
			strings.ToLower(countryCode),
			clientIP,
			expirationFor(now, throttleSettings.CapResets, timeZone)).Result()
		if err != nil {
			return err
		}

		result := _result.([]interface{})
		bytesIn, _ := result[0].(int64)
		bytesOut, _ := result[1].(int64)
		_countryCode := result[2]
		// In production it should never be nil but LedisDB (for unit testing)
		// has a bug which treats empty string as nil when `EvalSha`.
		if _countryCode == nil {
			countryCode = ""
		} else {
			countryCode = _countryCode.(string)
		}
		ttlSeconds := result[3].(int64)
		usage.Set(deviceID, countryCode, bytesIn+bytesOut, now, ttlSeconds)
	}
	return nil
}

func expirationFor(now time.Time, ttl throttle.CapInterval, timeZoneName string) int64 {
	tz, err := time.LoadLocation(timeZoneName)
	if err == nil {
		// adjust to given timeZone
		now = now.In(tz)
	}
	switch ttl {
	case throttle.Daily:
		tomorrow := now.AddDate(0, 0, 1)
		return time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 0, 0, 0, 0, now.Location()).Add(-1 * time.Nanosecond).Unix()
	case throttle.Monthly:
		nextMonth := now.AddDate(0, 1, 0)
		return time.Date(nextMonth.Year(), nextMonth.Month(), 1, 0, 0, 0, 0, now.Location()).Add(-1 * time.Nanosecond).Unix()
	}
	return 0
}
