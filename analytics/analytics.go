// package analytics provides logic for tracking popular sites accessed via this
// proxy server.
package analytics

import (
	"bytes"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"

	"github.com/getlantern/golog"
	"github.com/getlantern/http-proxy-lantern/devicefilter"
)

const (
	trackingId  = "UA-21815217-15" // corresponds to the Proxied Sites property
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
	next         http.Handler
	siteAccesses chan *siteAccess
	httpClient   *http.Client
}

func New(next http.Handler) *AnalyticsMiddleware {
	am := &AnalyticsMiddleware{
		next:         next,
		siteAccesses: make(chan *siteAccess, 10000),
		httpClient:   &http.Client{},
	}
	go am.submitToGoogle()
	return am
}

func (am *AnalyticsMiddleware) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	am.track(req)
	// For privacy, delete the DeviceId header before passing it along
	req.Header.Del(devicefilter.DeviceIdHeader)
	am.next.ServeHTTP(w, req)
}

func (am *AnalyticsMiddleware) track(req *http.Request) {
	am.siteAccesses <- &siteAccess{
		ip:       stripPort(req.RemoteAddr),
		clientId: req.Header.Get(devicefilter.DeviceIdHeader),
		site:     stripPort(req.Host),
	}
}

// submitToGoogle submits tracking information to Google Analytics on a
// goroutine to avoid blocking the processing of actual requests
func (am *AnalyticsMiddleware) submitToGoogle() {
	for sa := range am.siteAccesses {
		am.trackSession(sessionVals(sa))
	}
}

func sessionVals(sa *siteAccess) string {
	vals := make(url.Values, 0)

	// Version 1 of the API
	vals.Add("v", "1")
	// Our Google Tracking ID
	vals.Add("tid", trackingId)
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
	vals.Add("dp", sa.site)

	// Note the absence of session tracking. We don't have a good way to tell
	// when a session ends, so we don't bother with it.

	return vals.Encode()
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
