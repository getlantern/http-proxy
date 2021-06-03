// Package bbr provides support for BBR-based bandwidth estimation.
//
// Bandwidth estimates are provided to clients following the below protocol:
//
// 1. On every inbound connection, we interrogate BBR congestion control
//    parameters to determine the estimated bandwidth, extrapolate this to what
//    we would expected for a 2.5 MB transfer using a linear estimation based on
//    how much data has actually been transferred on the connection and then
//    maintain an exponential moving average (EMA) of these estimates per remote
//    (client) IP.
// 2. If a client includes HTTP header "X-BBR: <anything>", we include header
//    X-BBR-ABE: <EMA bandwidth in Mbps> in the HTTP response.
// 3. If a client includes HTTP header "X-BBR: clear", we clear stored estimate
//    data for the client's IP.
//
package bbr

import (
	"net"
	"net/http"

	"github.com/getlantern/http-proxy-lantern/v2/zerologger"
	"github.com/getlantern/proxy/filters"
)

const (
	nanosPerMilli = 1000000
)

var (
	log = zerologger.Named("bbrlistener")
)

type Middleware interface {
	filters.Filter

	// AddMetrics adds BBR metrics to the given response.
	AddMetrics(ctx filters.Context, req *http.Request, resp *http.Response)

	// Wrap wraps the given listener with support for BBR metrics.
	Wrap(l net.Listener) net.Listener

	// ABE returns an estimate of the available bandwidth in Mbps for the given
	// Context
	ABE(ctx filters.Context) float64

	// ProbeUpstream continuously probes the upstream URL and uses the BBR estimates
	// returned from upstream to determine the weakest link and adjust the ABE returned
	// in ABE().
	ProbeUpstream(url string)
}
