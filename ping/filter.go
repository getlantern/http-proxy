package ping

import (
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/getlantern/http-proxy-lantern/common"
	"github.com/getlantern/http-proxy/filters"
)

func (pm *pingMiddleware) Apply(resp http.ResponseWriter, req *http.Request, next filters.Next) error {
	log.Trace("In ping")
	pingSize := req.Header.Get(common.PingHeader)
	pingURL := req.Header.Get(common.PingURLHeader)
	if pingSize == "" && pingURL == "" {
		log.Trace("Bypassing ping")
		return next()
	}
	log.Trace("Processing ping")

	if pingURL != "" {
		// This is an old-style URL, simulate latency by sleeping
		pm.simulateLatency(pingURL)
		resp.WriteHeader(http.StatusOK)
		return filters.Stop()
	}

	// This is a new-style request, include ping summary in response body
	useGZ := false
	for _, encoding := range req.Header["Accept-Encoding"] {
		if encoding == "gzip" {
			useGZ = true
			break
		}
	}

	var summary []byte
	pm.summaryMX.RLock()
	if useGZ {
		summary = pm.summaryGZ
	} else {
		summary = pm.summary
	}
	pm.summaryMX.RUnlock()

	if summary == nil {
		// Don't have any ping data yet
		resp.WriteHeader(http.StatusResetContent)
		return filters.Stop()
	}

	resp.Header().Set("Content-Type", "application/json")
	if useGZ {
		resp.Header().Set("Content-Encoding", "gzip")
	}

	resp.WriteHeader(http.StatusOK)
	resp.Write(summary)

	return filters.Stop()
}

func (pm *pingMiddleware) simulateLatency(pingURL string) {
	pingOrigin := ""
	parsed, err := url.Parse(pingURL)
	if err != nil {
		pingOrigin = parsed.Host
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
}
