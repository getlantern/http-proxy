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
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/getlantern/golog"
	"github.com/getlantern/http-proxy-lantern/common"
	"github.com/getlantern/http-proxy/filters"
	"github.com/golang/groupcache/lru"
)

const (
	ApiEndpoint = `https://ssl.google-analytics.com/collect`
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

var (
	log = golog.LoggerFor("http-proxy-lantern.analytics")
)

// siteAccess holds information for tracking access to a site
type siteAccess struct {
	ip        string
	clientId  string
	site      string
	port      string
	userAgent string
}

type Options struct {
	TrackingID       string
	SamplePercentage float64
}

// analyticsMiddleware allows plugging popular sites tracking into the proxy's
// handler chain.
type analyticsMiddleware struct {
	*Options
	hostname     string
	siteAccesses chan *siteAccess
	httpClient   *http.Client
	dnsCache     *lru.Cache
}

func New(opts *Options) filters.Filter {
	hostname, err := os.Hostname()
	if err != nil {
		log.Errorf("Unable to determine hostname, will use '(direct))': %v", hostname)
		hostname = "(direct)"
	}
	log.Debugf("Will report analytics to Google as %v using hostname '%v', sampling %d percent of requests", opts.TrackingID, hostname, int(opts.SamplePercentage*100))
	am := &analyticsMiddleware{
		Options:      opts,
		hostname:     hostname,
		siteAccesses: make(chan *siteAccess, 1000),
		httpClient:   &http.Client{},
		dnsCache:     lru.New(2000),
	}
	go am.submitToGoogle()
	return am
}

func (am *analyticsMiddleware) Apply(w http.ResponseWriter, req *http.Request, next filters.Next) error {
	am.track(req)
	return next()
}

func (am *analyticsMiddleware) track(req *http.Request) {
	if rand.Float64() <= am.SamplePercentage {
		host, port, _ := net.SplitHostPort(req.Host)
		if hostExcluded(host) {
			return
		}
		if (port == "0" || port == "") && req.Method != http.MethodConnect {
			// Default port for HTTP
			port = "80"
		}
		select {
		case am.siteAccesses <- &siteAccess{
			ip:        stripPort(req.RemoteAddr),
			clientId:  req.Header.Get(common.DeviceIdHeader),
			site:      host,
			port:      port,
			userAgent: req.UserAgent(),
		}:
			// Submitted
		default:
			log.Debug("Site access request queue is full")
		}
	}
}

// submitToGoogle submits tracking information to Google Analytics on a
// goroutine to avoid blocking the processing of actual requests
func (am *analyticsMiddleware) submitToGoogle() {
	for sa := range am.siteAccesses {
		for _, site := range am.normalizeSite(sa.site, sa.port) {
			am.trackSession(am.sessionVals(sa, site, sa.port))
		}
	}
}

func (am *analyticsMiddleware) sessionVals(sa *siteAccess, site string, port string) string {
	vals := make(url.Values, 0)

	// Version 1 of the API
	vals.Add("v", "1")
	// Our Google Tracking ID
	vals.Add("tid", am.TrackingID)
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

	// Track custom port dimension
	vals.Add("cd1", port)

	log.Tracef("Tracking view to site: %v", site)
	vals.Add("dp", site)

	// Use the user-agent reported by the client
	vals.Add("ua", sa.userAgent)

	// Use the server's hostname as the campaign source so that we can track
	// activity per server
	vals.Add("cs", am.hostname)
	// Campaign medium and campaign name are required for campaign tracking to do
	// anything. We just fill them in with some dummy values.
	vals.Add("cm", "proxy")
	vals.Add("cn", "proxy")

	// Note the absence of session tracking. We don't have a good way to tell
	// when a session ends, so we don't bother with it.

	return vals.Encode()
}

func (am *analyticsMiddleware) normalizeSite(site string, port string) []string {
	domain := site
	result := make([]string, 0, 3)
	isIP := net.ParseIP(site) != nil
	if isIP {
		// This was an ip, do a reverse lookup
		cached, found := am.dnsCache.Get(site)
		if !found {
			names, err := net.LookupAddr(site)
			if err != nil {
				log.Debugf("Unable to perform reverse DNS lookup for %v: %v", site, err)
				cached = site
			} else {
				name := names[0]
				if name != "" && name[len(name)-1] == '.' {
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
	if domain != "" && domain != site {
		// If original site is not the same as domain, track that too
		result = append(result, domain)
		// Also track just the last two portions of the domain name
		parts := strings.Split(domain, ".")
		if len(parts) > 1 {
			result = append(result, "/generated/"+strings.Join(parts[len(parts)-2:], "."))
		}
	}

	switch port {
	case "80":
		result = append(result, "/protocol/http")
	case "443":
		result = append(result, "/protocol/https")
	case "0", "":
		result = append(result, "/protocol/unknown")
	default:
		result = append(result, "/protocol/other")
	}
	return result
}

func (am *analyticsMiddleware) trackSession(args string) {
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

func hostExcluded(host string) bool {
	return host == "ping-chained-server" ||
		host == "config.getiantem.org" ||
		host == "logs-01.loggly.com" ||
		host == "borda.getlantern.org" ||
		host == "www.google-analytics.com"
}
