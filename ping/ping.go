// package ping provides a ping-like service that gives insight into the
// performance of this proxy.
package ping

import (
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/getlantern/go-ping"
	"github.com/getlantern/golog"
	"github.com/getlantern/http-proxy-lantern/common"
	"github.com/getlantern/http-proxy/filters"
)

const (
	pingInterval = 5 * time.Minute
)

var (
	log = golog.LoggerFor("http-proxy-lantern.ping")

	// Reasonable assumptions about stats until we know more from running checks
	defaultStats = &stats{
		rtt: 10 * time.Millisecond,
		plr: 0,
	}
)

type stats struct {
	origin string
	rtt    time.Duration
	plr    float64
}

type pingMiddleware struct {
	pinger        *ping.Pinger
	statsByOrigin map[string]*stats
	httpClient    *http.Client
	mx            sync.RWMutex
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

	pm.mx.RLock()
	statsByOrigin := pm.statsByOrigin
	pm.mx.RUnlock()

	var s *stats
	// Go through subsub.sub.domain.tld, stripping away subdomains until we get
	// a result or run out of domain.
	parts := strings.Split(strings.ToLower(pingOrigin), ".")
	for {
		if len(parts) < 2 {
			break
		}
		origin := strings.Join(parts, ".")
		log.Debugf("Checking %v", origin)
		s = statsByOrigin[origin]
		if s != nil {
			log.Debugf("Got data for %v", origin)
			break
		}
		parts = parts[1:]
	}

	if s == nil {
		// Use googlevideo.com by default
		s = statsByOrigin["googlevideo.com"]
	}

	if s == nil {
		s = defaultStats
	}

	if pingURL != "" {
		// This is an old-style ping, simulate latency by sleeping
		time.Sleep(s.rtt * 50)
	}
	w.Header().Set(common.PingRTTHeader, fmt.Sprint(s.rtt))
	w.Header().Set(common.PingPLRHeader, fmt.Sprint(s.plr))
	w.WriteHeader(http.StatusOK)
	return filters.Stop()
}

func New() (filters.Filter, error) {
	// Run privileged on Windows where this doesn't require root and where udp
	// encapsulation doesn't work
	privileged := runtime.GOOS == "windows"
	pinger, err := ping.NewPinger(privileged)
	if err != nil {
		return nil, fmt.Errorf("Unable to initialize pinger: %v", err)
	}

	pm := &pingMiddleware{
		pinger:        pinger,
		statsByOrigin: make(map[string]*stats, len(origins)),
		httpClient: &http.Client{
			Transport: &http.Transport{
				DisableKeepAlives: true,
			},
		},
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
		log.Debugf("ping results for %v  rtt: %4dms  plr: %3.2f%%", s.origin, s.rtt.Nanoseconds()/1000000, s.plr)
		pm.mx.Lock()
		// Copy stats map
		statsByOrigin := make(map[string]*stats, len(origins))
		for key, value := range pm.statsByOrigin {
			statsByOrigin[key] = value
		}
		// Update stats for this origin
		statsByOrigin[s.origin] = s
		pm.statsByOrigin = statsByOrigin
		pm.mx.Unlock()
	}
}

func (pm *pingMiddleware) doUpdateTimings(resultCh chan *stats) {
	// With current number of origins and settings, each loop of this will take
	// between 10 and 30 minutes, so there's no need to have an artificial pause.

}

func (pm *pingMiddleware) pingOrigin(origin string, resultCh chan *stats) {
	for {
		log.Debugf("pinging %v", origin)
		st, err := pm.pinger.Ping(origin,
			// Ping 100 times to get a decent resolution
			100,
			// Ping every 200 milliseconds, which is the lowest amount that OS ping
			// command allows without root (to avoid ICMP flooding)
			200*time.Millisecond,
			// Don't let pinging take more than 1 minute total
			1*time.Minute)
		if err != nil {
			log.Errorf("Error pinging %v: %v", origin, err)
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
