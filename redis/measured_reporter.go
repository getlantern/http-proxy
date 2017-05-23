package redis

import (
	"math/rand"
	"strconv"
	"time"

	"gopkg.in/redis.v3"

	"github.com/getlantern/http-proxy-lantern/geo"
	"github.com/getlantern/http-proxy-lantern/usage"
	"github.com/getlantern/http-proxy/listeners"
	"github.com/getlantern/measured"
)

const script = `
	local clientKey = KEYS[1]

	local bytesIn = redis.call("hincrby", clientKey, "bytesIn", ARGV[1])
	local bytesOut = redis.call("hincrby", clientKey, "bytesOut", ARGV[2])
	redis.call("hsetnx", clientKey, "clientIP", ARGV[3])
	redis.call("hsetnx", clientKey, "countryCode", ARGV[4])
	local countryCode = redis.call("hget", clientKey, "countryCode")

	redis.call("expireat", clientKey, ARGV[5])
	return {bytesIn, bytesOut, countryCode}
`

var (
	keysExpiration = time.Hour * 24 * 31

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
					// ignore
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
			// if log.IsTraceEnabled() {
			log.Debugf("Submitting %d stats", len(statsByDeviceID))
			// }
			err := submit(rc, statsByDeviceID)
			if err != nil {
				log.Errorf("Unable to submit stats: %v", err)
			}
			// Reset stats
			statsByDeviceID = make(map[string]*statsAndIP)
		}
	}
}

func submit(rc *redis.Client, statsByDeviceID map[string]*statsAndIP) error {
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
		_result, err := rc.Eval(script, []string{clientKey}, []string{
			strconv.Itoa(stats.RecvTotal),
			strconv.Itoa(stats.SentTotal),
			stats.ip,
			countryCode,
			endOfThisMonth,
		}).Result()
		if err != nil {
			return err
		}

		result := _result.([]interface{})
		bytesIn, _ := result[0].(int64)
		bytesOut, _ := result[1].(int64)
		countryCode = result[2].(string)
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
