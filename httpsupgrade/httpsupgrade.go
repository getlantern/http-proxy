// Package httpsupgrade performs several functions. First, it upgrades uncoming HTTP requests to
// HTTPS for hitting our own services. This is necessary because we need to add special headers
// to those requests, but we cannot do that if they're over TLS. Note that the incoming requests
// are all wrapped in the encryption of the incoming transport, however.
//
// This pacakge also short circuits the normal proxy processing of requests to make direct
// HTTP/2 connections to specific domains such as Lantern internal servers. This saves
// a significant amount of CPU in reducing TLS client handshakes but also makes these
// requests more efficient through the use of persistent connections and HTTP/2's multiplexing
// and reduced headers.
//
// It is worth noting this technique only works over HTTP/2 if the upstream provider, such as
// Cloudflare, supports it. Otherwise it will use regular HTTP and will not benefit to the
// same degree from all of the above benefits, although it will still likely be an improvement
// due to the use of persistent connections.
package httpsupgrade

import (
	"net"
	"net/http"
	"time"

	"github.com/getlantern/golog"
	"github.com/getlantern/http-proxy-lantern/common"
	"github.com/getlantern/http-proxy-lantern/domains"
	"github.com/getlantern/proxy/filters"
)

type httpsUpgrade struct {
	httpClient            *http.Client
	log                   golog.Logger
	configServerAuthToken string
}

// NewHTTPSUpgrade creates a new request filter for rewriting requests to HTTPS services..
func NewHTTPSUpgrade(configServerAuthToken string) filters.Filter {
	return &httpsUpgrade{
		httpClient: &http.Client{
			Transport: &http.Transport{
				IdleConnTimeout: 4 * time.Minute,
			},
		},
		log:                   golog.LoggerFor("httpsUpgrade"),
		configServerAuthToken: configServerAuthToken,
	}
}

// Apply implements the filters.Filter interface for HTTP request processing.
func (h *httpsUpgrade) Apply(ctx filters.Context, req *http.Request, next filters.Next) (*http.Response, filters.Context, error) {
	if req.Method == "CONNECT" {
		return next(ctx, req)
	}
	if cfg := domains.ConfigForRequest(req); cfg.RewriteToHTTPS {
		if cfg.AddConfigServerHeaders {
			h.addConfigServerHeaders(req)
		}
		return h.rewrite(ctx, cfg.Host, req)
	}
	return next(ctx, req)
}

func (h *httpsUpgrade) addConfigServerHeaders(req *http.Request) {
	if h.configServerAuthToken == "" {
		h.log.Error("No config server auth token?")
		return
	}
	req.Header.Set(common.CfgSvrAuthTokenHeader, h.configServerAuthToken)
	ip, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		h.log.Errorf("Unable to split host from '%s': %s", req.RemoteAddr, err)
	} else {
		req.Header.Set(common.CfgSvrClientIPHeader, ip)
	}
}

func (h *httpsUpgrade) rewrite(ctx filters.Context, host string, req *http.Request) (*http.Response, filters.Context, error) {
	req.Host = host + ":443"
	req.URL.Host = req.Host
	req.URL.Scheme = "https"
	h.log.Debugf("Rewrote request with URL %#v to HTTPS", req.URL)
	// Make sure the request stays open.
	req.Close = false

	// The request URI is populated in the request to the proxy but raises an error if populated
	// in outgoing client requests.
	req.RequestURI = ""

	res, err := h.httpClient.Do(req)
	if err != nil {
		h.log.Errorf("Error short circuiting with HTTP/2 with req %#v, %v", req, err)
		return res, ctx, err
	}
	return filters.ShortCircuit(ctx, req, res)
}
