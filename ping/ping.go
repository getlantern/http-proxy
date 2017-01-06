// package ping provides a ping-like service that gives insight into the
// performance of this proxy.
package ping

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/getlantern/ema"
	"github.com/getlantern/go-ping"
	"github.com/getlantern/golog"
	"github.com/getlantern/http-proxy/filters"
	"github.com/getlantern/ops"
)

const (
	// assume maximum segment size of 1460 for Mathis throughput calculation
	mss = 1460

	// evolve exponential moving averages fairly quickly
	emaAlpha = 0.5

	nanosPerMilli = 1000000
)

var (
	log = golog.LoggerFor("http-proxy-lantern.ping")

	// Reasonable assumptions about stats until we know more from running checks
	defaultStats = &stats{
		rtt: 10 * time.Millisecond,
		plr: 0,
	}

	defaultEMAStats = &emaStats{
		rtt: ema.NewDuration(defaultStats.rtt, emaAlpha),
		plr: ema.New(defaultStats.plr, emaAlpha),
	}
)

type stats struct {
	origin string
	rtt    time.Duration
	plr    float64
}

type emaStats struct {
	rtt *ema.EMA
	plr *ema.EMA
}

// mathisThroughput estimates throughput using the Mathis equation
// See https://www.switch.ch/network/tools/tcp_throughput/?do+new+calculation=do+new+calculation
// for example.
// Returns value in Mbps
func (s *emaStats) mathisThroughput() float64 {
	rtt := s.rtt.GetDuration()
	plr := s.plr.Get() / 100
	if plr == 0 {
		// Assume small but measurable packet loss
		// I came up with this number by comparing the result for
		// download.thinkbroadband.com to actual download speeds.
		plr = 0.000005
	}
	return 8 * (mss / rtt.Seconds()) * (1.0 / math.Sqrt(plr)) / nanosPerMilli
}

type pingMiddleware struct {
	pinger          *ping.Pinger
	statsByOrigin   map[string]*emaStats
	summary         []byte
	summaryGZ       []byte
	summaryMX       sync.RWMutex
	httpClient      *http.Client
	doReportToBorda func(map[string]float64, map[string]interface{}) error
}

func New(reportToBorda func(map[string]float64, map[string]interface{}) error) (filters.Filter, error) {
	// Run privileged on Windows where this doesn't require root and where udp
	// encapsulation doesn't work. Also run as privileged on linux where we have
	// to deal with a firewall.
	privileged := runtime.GOOS == "windows" || runtime.GOOS == "linux"
	pinger, err := ping.NewPinger(privileged)
	if err != nil {
		return nil, fmt.Errorf("Unable to initialize pinger: %v", err)
	}

	sbo := make(map[string]*emaStats, len(origins))
	for _, origin := range origins {
		sbo[origin] = &emaStats{
			rtt: ema.NewDuration(defaultStats.rtt, emaAlpha),
			plr: ema.New(defaultStats.plr, emaAlpha),
		}
	}

	pm := &pingMiddleware{
		pinger:        pinger,
		statsByOrigin: sbo,
		httpClient: &http.Client{
			Transport: &http.Transport{
				DisableKeepAlives: true,
			},
		},
		doReportToBorda: reportToBorda,
	}
	go pm.ping()

	return pm, nil
}

func (pm *pingMiddleware) ping() {
	resultCh := make(chan *stats, len(origins))
	for _, origin := range origins {
		go pm.pingOrigin(origin, resultCh)
	}

	for s := range resultCh {
		es := pm.statsByOrigin[s.origin]
		es.rtt.UpdateDuration(s.rtt)
		es.plr.Update(s.plr)
		log.Debugf("ping stats for %v  rtt: %4v  plr: %3.2f%%  tput: %5.0fMpbs", s.origin, es.rtt.GetDuration(), es.plr.Get(), es.mathisThroughput())

		// Generate summary
		summary, summaryGZ, err := pm.generateSummary()
		if err != nil {
			log.Error(err)
			continue
		}
		pm.summaryMX.Lock()
		pm.summary = summary
		pm.summaryGZ = summaryGZ
		pm.summaryMX.Unlock()

		if pm.doReportToBorda != nil {
			pm.reportToBorda(s.origin, es)
		}
	}
}

func (pm *pingMiddleware) generateSummary() ([]byte, []byte, error) {
	statsByOrigin := make(map[string]map[string]interface{}, len(pm.statsByOrigin))
	for origin, st := range pm.statsByOrigin {
		statsByOrigin[origin] = map[string]interface{}{
			"rtt":  float64(st.rtt.GetDuration().Nanoseconds()) / nanosPerMilli,
			"plr":  st.plr.Get(),
			"tput": st.mathisThroughput(),
		}
	}
	summary, err := json.Marshal(statsByOrigin)
	if err != nil {
		return nil, nil, fmt.Errorf("Unable to marshal stats to json: %v", err)
	}

	buf := &bytes.Buffer{}
	w, err := gzip.NewWriterLevel(buf, gzip.BestCompression)
	if err != nil {
		return nil, nil, fmt.Errorf("Unable to create gzip writer: %v", err)
	}
	_, err = w.Write(summary)
	if err != nil {
		return nil, nil, fmt.Errorf("Unable to write gzip: %v", err)
	}
	err = w.Close()
	if err != nil {
		return nil, nil, fmt.Errorf("Unable to finish writing gzip: %v", err)
	}

	summaryGZ := buf.Bytes()
	return summary, summaryGZ, nil
}

func (pm *pingMiddleware) reportToBorda(origin string, es *emaStats) {
	dims := ops.AsMap(nil, true)
	dims["origin_host"] = origin
	dims["op"] = "ping_origin"
	err := pm.doReportToBorda(map[string]float64{
		"ping_rtt":  es.rtt.GetDuration().Seconds(),
		"ping_plr":  es.plr.Get(),
		"ping_tput": es.mathisThroughput(),
	}, dims)
	if err != nil {
		log.Errorf("Unable to submit to borda: %v", err)
	}
}

func (pm *pingMiddleware) pingOrigin(origin string, resultCh chan *stats) {
	log.Debugf("monitoring %v", origin)
	pm.pinger.Loop(origin,
		// Start with a batch of 4 to get a quick initial result
		4,
		// Use batches of 256 to get a decent resolution
		256,
		// Ping every 200 milliseconds, which is the lowest amount that OS ping
		// command allows without root (to avoid ICMP flooding)
		200*time.Millisecond,
		// Set timeout assuming that RTT will be less than 400ms
		400*time.Millisecond,
		func(st *ping.Statistics, err error) bool {
			if err != nil {
				log.Errorf("Error pinging %v: %v", origin, err)
			} else if st.PacketsSent == 0 {
				log.Debugf("No packets sent to %v, ignoring results", origin)
			} else {
				resultCh <- &stats{
					origin: origin,
					rtt:    st.AvgRtt,
					plr:    st.PacketLoss,
				}
			}

			// Always continue looping
			return true
		})
}

// adds randomization to make requests less distinguishable on the network.
func randomize(d time.Duration) time.Duration {
	return time.Duration((d.Nanoseconds() / 2) + rand.Int63n(d.Nanoseconds()))
}

var (
	// Hardcoded list of origins to ping, based on top sites reported in borda
	origins = []string{
		"facebook.com",
		"fbcdn.net",
		"mtalk.google.com",
		"clients1.google.com",
		"clients2.google.com",
		"clients3.google.com",
		"clients4.google.com",
		"clients5.google.com",
		"clients6.google.com",
		"safebrowsing.google.com",
		"google.com",
		"google-analytics.com",
		"gstatic.com",
		"i.ytimg.com",
		"i1.ytimg.com",
		"i9.ytimg.com",
		"s.ytimg.com",
		"youtube.com",
		"googlevideo.com",
		"twitter.com",
		"abs.twimg.com",
		"pbs.twimg.com",
		"mail.yahoo.com",
		"yahoo.com",
		"149.154.167.91",
		"tumblr.com",
		"dropbox.com",
		"fb-s-a-a.akamaihd.net",
		"fb-s-b-a.akamaihd.net",
		"fb-s-c-a.akamaihd.net",
		"fb-s-d-a.akamaihd.net",
		"fbexternal-a.akamaihd.net",
		"steamcommunity-a.akamaihd.net",
	}
)
