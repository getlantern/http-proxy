// package ping provides a ping-like service that gives insight into the
// performance of this proxy.
package ping

import (
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"runtime"
	"strings"
	"time"

	"github.com/getlantern/ema"
	"github.com/getlantern/go-ping"
	"github.com/getlantern/golog"
	"github.com/getlantern/http-proxy-lantern/common"
	"github.com/getlantern/http-proxy/filters"
)

const (
	// Ping 200 times to get a decent resolution
	pingBatch = 200

	// Ping every 200 milliseconds, which is the lowest amount that OS ping
	// command allows without root (to avoid ICMP flooding)
	pingInterval = 200 * time.Millisecond

	// assume maximum segment size of 1460 for Mathis throughput calculation
	mss = 1460

	// evolve exponential moving averages fairly quickly
	emaAlpha = 0.5
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
// Returns value in Kbps
func (s *emaStats) mathisThroughput() float64 {
	rtt := s.rtt.GetDuration()
	plr := s.plr.Get() / 100
	if plr == 0 {
		// Assume small but measurable packet loss
		// I came up with this number by comparing the result for
		// download.thinkbroadband.com to actual download speeds.
		plr = 0.000005
	}
	return 8 * (mss / rtt.Seconds()) * (1.0 / math.Sqrt(plr)) / 1000
}

type pingMiddleware struct {
	pinger        *ping.Pinger
	statsByOrigin map[string]*emaStats
	httpClient    *http.Client
}

func New() (filters.Filter, error) {
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
	}
	go pm.ping()

	return pm, nil
}

func (pm *pingMiddleware) Apply(w http.ResponseWriter, req *http.Request, next filters.Next) error {
	log.Trace("In ping")
	pingSize := req.Header.Get(common.PingHeader)
	pingURL := req.Header.Get(common.PingURLHeader)
	pingOrigin := req.Header.Get(common.PingOriginHeader)
	if pingSize == "" && pingURL == "" && pingOrigin == "" {
		log.Trace("Bypassing ping")
		return next()
	}
	log.Trace("Processing ping")

	if pingOrigin == "" && pingURL != "" {
		parsed, err := url.Parse(pingURL)
		if err != nil {
			pingOrigin = parsed.Host
		}
	}

	var s *emaStats
	// Go through subsub.sub.domain.tld, stripping away subdomains until we get
	// a result or run out of domain.
	parts := strings.Split(strings.ToLower(pingOrigin), ".")
	for {
		if len(parts) < 2 {
			break
		}
		origin := strings.Join(parts, ".")
		log.Debugf("Checking %v", origin)
		s = pm.statsByOrigin[origin]
		if s != nil {
			log.Debugf("Got data for %v", origin)
			break
		}
		parts = parts[1:]
	}

	if s == nil {
		// Use googlevideo.com by default
		s = pm.statsByOrigin["googlevideo.com"]
	}

	if s == nil {
		s = defaultEMAStats
	}

	if pingURL != "" {
		// This is an old-style ping, simulate latency by sleeping
		time.Sleep(s.rtt.GetDuration() * 50)
	}
	w.Header().Set(common.PingThroughputHeader, fmt.Sprint(s.mathisThroughput()))
	w.WriteHeader(http.StatusOK)
	return filters.Stop()
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
		log.Debugf("ping stats for %v  rtt: %4v  plr: %3.2f%%  tput: %5.0fKpbs", s.origin, es.rtt.GetDuration(), es.plr.Get(), es.mathisThroughput())
	}
}

func (pm *pingMiddleware) doUpdateTimings(resultCh chan *stats) {
	// With current number of origins and settings, each loop of this will take
	// between 10 and 30 minutes, so there's no need to have an artificial pause.

}

func (pm *pingMiddleware) pingOrigin(origin string, resultCh chan *stats) {
	log.Debugf("monitoring %v", origin)
	for {
		st, err := pm.pinger.Ping(origin,
			pingBatch,
			pingInterval,
			// Don't let pinging take more than twice pingBatch * pingInterval
			time.Duration(pingBatch)*pingInterval*2)
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
		time.Sleep(randomize(pingInterval))
	}
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
