// Package httpsrewriter is responsible for rewriting incoming requests over HTTP to outgoing services running HTTPS.
// This is necesssary in some cases where we need to add custom headers to incoming requests for things like
// authentication. For incomging CONNECT requests, we can add custom headers, but CDNs upstread typically strip out
// any extra headers from CONNECTs. As a result, the incoming requests often need to happen over HTTP (wrapped, of
// course, in whatever transport this server uses), and are upgraded here to HTTPS.
package httpsrewriter

import (
	"net"
	"net/http"
	"time"

	"github.com/getlantern/golog"
	"github.com/getlantern/http-proxy-lantern/common"
	"github.com/getlantern/http-proxy-lantern/domains"
	"github.com/getlantern/proxy/filters"
)

type rewriter struct {
	httpClient      *http.Client
	cfgSvrAuthToken string
	log             golog.Logger
}

// NewRewriter creates a new request filter for rewriting requests to HTTPS services..
func NewRewriter(cfgSvrAuthToken string) filters.Filter {
	return &rewriter{
		httpClient: &http.Client{
			Transport: &http.Transport{
				IdleConnTimeout: 4 * time.Minute,
			},
		},
		cfgSvrAuthToken: cfgSvrAuthToken,
		log:             golog.LoggerFor("httpsRewritter"),
	}
}

// Apply implements the filters.Filter interface for HTTP request processing.
func (r *rewriter) Apply(ctx filters.Context, req *http.Request, next filters.Next) (*http.Response, filters.Context, error) {
	if req.Method == "CONNECT" {
		return next(ctx, req)
	}
	if cfg := domains.ConfigForRequest(req); cfg.RewriteToHTTPS {
		r.rewrite(cfg.Host, req)
		if cfg.AddConfigServerHeaders {
			r.addConfigServerHeaders(req)
		}
		res, err := r.httpClient.Do(req)
		if err != nil {
			r.log.Debugf("Error fetching with req %#v, %v", req, err)
			return next(ctx, req)
		}
		r.log.Debugf("Response: %#v", res)
		return res, ctx, nil
	}
	return next(ctx, req)
}

func (r *rewriter) rewrite(host string, req *http.Request) {
	r.log.Debugf("Rewriting request to HTTPS: %#v", req)
	r.log.Debugf("Rewriting request to HTTPS with URL: %#v", req.URL)
	req.Host = host + ":443"
	req.URL.Host = req.Host
	req.URL.Scheme = "https"
	req.Close = false

	// The request URI is populated in the request to the proxy but raises an error if populated in outgoing client
	// requests.
	req.RequestURI = ""
	r.log.Debugf("Rewrote request to HTTPS: %#v", req)
	r.log.Debugf("Rewrote request to HTTPS with URL: %#v", req.URL)

	testReq, _ := http.NewRequest("GET", "https://config.getiantem.org/proxies.yaml.gz", nil)

	r.log.Debugf("Test request to HTTPS: %#v", testReq)
	r.log.Debugf("Test request to HTTPS with URL: %#v", testReq.URL)
}

func (r *rewriter) addConfigServerHeaders(req *http.Request) {
	if r.cfgSvrAuthToken == "" {
		r.log.Error("No config server auth token")
		return
	}
	req.Header.Set(common.CfgSvrAuthTokenHeader, r.cfgSvrAuthToken)
	ip, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		r.log.Errorf("Unable to split host from '%s': %s", req.RemoteAddr, err)
		return
	}
	req.Header.Set(common.CfgSvrClientIPHeader, ip)
	r.log.Debugf("Adding header to config-server request from %s to %s", ip, req.Host)
}
