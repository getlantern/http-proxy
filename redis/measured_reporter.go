package redis

import (
	"math/rand"
	"strconv"
	"time"

	"gopkg.in/redis.v5"

	"github.com/getlantern/golog"
	"github.com/getlantern/http-proxy-lantern/geo"
	"github.com/getlantern/http-proxy-lantern/usage"
	"github.com/getlantern/http-proxy/listeners"
	"github.com/getlantern/measured"
)

const script = `
	local clientKey = KEYS[1]

	local bytesIn = redis.call("hincrby", clientKey, "bytesIn", ARGV[1])
	local bytesOut = redis.call("hincrby", clientKey, "bytesOut", ARGV[2])
	local countryCode = redis.call("hget", clientKey, "countryCode")
	-- note that we use an if instead of just calling hsetnx because LedisDB (unit
	-- testing) doesn't support hsetnx
	if not countryCode then
		redis.call("hset", clientKey, "countryCode", ARGV[3])
		-- record the IP on which we based the countryCode for auditing
		redis.call("hset", clientKey, "clientIP", ARGV[4])
		countryCode = ARGV[3]
		redis.call("expireat", clientKey, ARGV[5])
	end

	return {bytesIn, bytesOut, countryCode}
`

var (
	log = golog.LoggerFor("redis")

	geoLookup = geo.New(1000000)
)

func init() {
	go trackGeoStats()
}

type statsAndContext struct {
	ctx   map[string]interface{}
	stats *measured.Stats
}

func NewMeasuredReporter(rc *redis.Client, reportInterval time.Duration) listeners.MeasuredReportFN {
	// Provide some buffering so that we don't lose data while submitting to Redis
	statsCh := make(chan *statsAndContext, 10000)
	go reportPeriodically(rc, reportInterval, statsCh)
	return func(ctx map[string]interface{}, stats *measured.Stats, deltaStats *measured.Stats, final bool) {
		select {
		case statsCh <- &statsAndContext{ctx, deltaStats}:
			// submitted successfully
		default:
			// data lost, probably because Redis submission is taking longer than expected
		}
	}
}

type statsAndIP struct {
	*measured.Stats
	ip string
}

func reportPeriodically(rc *redis.Client, reportInterval time.Duration, statsCh chan (*statsAndContext)) {
	// randomize the interval to evenly distribute traffic to reporting Redis.
	randomized := time.Duration(reportInterval.Nanoseconds()/2 + rand.Int63n(reportInterval.Nanoseconds()))
	log.Debugf("Will report data usage to Redis every %v", randomized)
	ticker := time.NewTicker(randomized)
	statsByDeviceID := make(map[string]*statsAndIP)
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
				_clientIP := sac.ctx["client_ip"]
				if _clientIP == nil {
					log.Error("Missing client_ip in context, this shouldn't happen. Ignoring.")
					continue
				}
				clientIP := _clientIP.(string)
				existing = &statsAndIP{
					Stats: sac.stats,
					ip:    clientIP,
				}
				statsByDeviceID[deviceID] = existing
			} else {
				existing.SentTotal += sac.stats.SentTotal
				existing.RecvTotal += sac.stats.RecvTotal
			}
		case <-ticker.C:
			if log.IsTraceEnabled() {
				log.Tracef("Submitting %d stats", len(statsByDeviceID))
			}
			if scriptSHA == "" {
				var err error
				scriptSHA, err = rc.ScriptLoad(script).Result()
				if err != nil {
					log.Errorf("Unable to load script, skip submitting stats: %v", err)
					continue
				}
			}

			err := submit(rc, scriptSHA, statsByDeviceID)
			if err != nil {
				log.Errorf("Unable to submit stats: %v", err)
			}
			// Reset stats
			statsByDeviceID = make(map[string]*statsAndIP)
		}
	}
}

func submit(rc *redis.Client, scriptSHA string, statsByDeviceID map[string]*statsAndIP) error {
	now := time.Now()
	nextMonth := now.Month() + 1
	nextYear := now.Year()
	if nextMonth > time.December {
		nextMonth = time.January
		nextYear++
	}
	beginningOfNextMonth := time.Date(nextYear, nextMonth, 1, 0, 0, 0, 0, now.Location())
	endOfThisMonth := strconv.Itoa(int(beginningOfNextMonth.Add(-1 * time.Nanosecond).Unix()))
	for deviceID, stats := range statsByDeviceID {
		clientKey := "_client:" + deviceID
		countryCode := geoLookup.CountryCode(stats.ip)
		_result, err := rc.EvalSha(scriptSHA, []string{clientKey},
			strconv.Itoa(stats.RecvTotal),
			strconv.Itoa(stats.SentTotal),
			countryCode,
			stats.ip,
			endOfThisMonth).Result()
		if err != nil {
			return err
		}

		result := _result.([]interface{})
		bytesIn, _ := result[0].(int64)
		bytesOut, _ := result[1].(int64)
		_countryCode := result[2]
		if _countryCode == nil {
			countryCode = ""
		} else {
			countryCode = _countryCode.(string)
		}
		usage.Set(deviceID, countryCode, bytesIn+bytesOut, now)
	}
	return nil
}

func trackGeoStats() {
	for {
		time.Sleep(1 * time.Minute)
		log.Debugf("Geo - Cache Size: %d   Cache Hits: %d   Network Lookups: %d   Network Lookup Errors: %d",
			geoLookup.CacheSize(),
			geoLookup.CacheHits(),
			geoLookup.NetworkLookups(),
			geoLookup.NetworkLookupErrors())
	}
}
