// package analytics provides logic for tracking popular sites accessed via this
// proxy server.
package analytics

import (
	"bytes"
	"math/rand"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"

	"github.com/getlantern/golog"
	"github.com/getlantern/http-proxy-lantern/common"
	"github.com/golang/groupcache/lru"
)

const (
	ApiEndpoint = `https://ssl.google-analytics.com/collect`
)

var (
	log = golog.LoggerFor("http-proxy-lantern.analytics")
)

// siteAccess holds information for tracking access to a site
type siteAccess struct {
	ip       string
	clientId string
	site     string
}

// AnalyticsMiddleware allows plugging popular sites tracking into the proxy's
// handler chain.
type AnalyticsMiddleware struct {
	trackingId       string
	samplePercentage float64
	next             http.Handler
	siteAccesses     chan *siteAccess
	httpClient       *http.Client
	dnsCache         *lru.Cache
}

func New(trackingId string, samplePercentage float64, next http.Handler) *AnalyticsMiddleware {
	am := &AnalyticsMiddleware{
		trackingId:       trackingId,
		samplePercentage: samplePercentage,
		next:             next,
		siteAccesses:     make(chan *siteAccess, 1000),
		httpClient:       &http.Client{},
		dnsCache:         lru.New(50000),
	}
	go am.submitToGoogle()
	return am
}

func (am *AnalyticsMiddleware) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	am.track(req)
	am.next.ServeHTTP(w, req)
}

func (am *AnalyticsMiddleware) track(req *http.Request) {
	if rand.Float64() <= am.samplePercentage {
		select {
		case am.siteAccesses <- &siteAccess{
			ip:       stripPort(req.RemoteAddr),
			clientId: req.Header.Get(common.DeviceIdHeader),
			site:     stripPort(req.Host),
		}:
			// Submitted
		default:
			log.Debug("Site access request queue is full")
		}
	}
}

// submitToGoogle submits tracking information to Google Analytics on a
// goroutine to avoid blocking the processing of actual requests
func (am *AnalyticsMiddleware) submitToGoogle() {
	for sa := range am.siteAccesses {
		for _, site := range am.normalizeSite(sa.site) {
			am.trackSession(am.sessionVals(sa, site))
		}
	}
}

func (am *AnalyticsMiddleware) sessionVals(sa *siteAccess, site string) string {
	vals := make(url.Values, 0)

	// Version 1 of the API
	vals.Add("v", "1")
	// Our Google Tracking ID
	vals.Add("tid", am.trackingId)
	// The client's ID (Lantern DeviceID, which is Base64 encoded 6 bytes from mac
	// address)
	vals.Add("cid", sa.clientId)

	// Override the users IP so we get accurate geo data.
	// vals.Add("uip", ip)
	vals.Add("uip", sa.ip)
	// Make call to anonymize the user's IP address -- basically a policy thing where
	// Google agrees not to store it.
	vals.Add("aip", "1")

	// Track this as a page view
	vals.Add("t", "pageview")

	// Do a reverse DNS lookup if necessary
	log.Tracef("Tracking view to site: %v", site)
	vals.Add("dp", site)

	// Note the absence of session tracking. We don't have a good way to tell
	// when a session ends, so we don't bother with it.

	return vals.Encode()
}

func (am *AnalyticsMiddleware) normalizeSite(site string) []string {
	domain := site
	result := make([]string, 0, 3)
	isIP := net.ParseIP(site) != nil
	if isIP {
		// This was an ip, do a reverse lookup
		cached, found := am.dnsCache.Get(site)
		if !found {
			names, err := net.LookupAddr(site)
			if err != nil {
				log.Errorf("Unable to perform reverse DNS lookup for %v: %v", site, err)
				cached = site
			} else {
				name := names[0]
				if name[len(name)-1] == '.' {
					// Strip trailing period
					name = name[:len(name)-1]
				}
				cached = name
			}
			am.dnsCache.Add(site, cached)
		}
		domain = cached.(string)
	}

	result = append(result, site)
	if domain != site {
		// If original site is not the same as domain, track that too
		result = append(result, domain)
		// Also track just the last two portions of the domain name
		parts := strings.Split(domain, ".")
		if len(parts) > 1 {
			result = append(result, "/generated/"+strings.Join(parts[len(parts)-2:], "."))
		}
	}

	return result
}

func (am *AnalyticsMiddleware) trackSession(args string) {
	r, err := http.NewRequest("POST", ApiEndpoint, bytes.NewBufferString(args))

	if err != nil {
		log.Errorf("Error constructing GA request: %s", err)
		return
	}

	r.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Add("Content-Length", strconv.Itoa(len(args)))

	if log.IsTraceEnabled() {
		if req, err := httputil.DumpRequestOut(r, true); err != nil {
			log.Errorf("Could not dump request: %v", err)
		} else {
			log.Tracef("Full analytics request: %v", string(req))
		}
	}

	resp, err := am.httpClient.Do(r)
	if err != nil {
		log.Errorf("Could not send HTTP request to GA: %s", err)
		return
	}
	log.Tracef("Successfully sent request to GA: %s", resp.Status)
	if err := resp.Body.Close(); err != nil {
		log.Debugf("Unable to close response body: %v", err)
	}
}

// stripPort strips the port from an address by removing everything after the
// last colon
func stripPort(addr string) string {
	lastColon := strings.LastIndex(addr, ":")
	if lastColon == -1 {
		// No colon, use addr as is
		return addr
	}
	return addr[:lastColon]
}
