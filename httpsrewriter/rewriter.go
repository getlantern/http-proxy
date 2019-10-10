// Package httpsrewriter is responsible for rewriting incoming requests over HTTP to outgoing services running HTTPS.
// This is necesssary in some cases where we need to add custom headers to incoming requests for things like
// authentication. For incomging CONNECT requests, we can add custom headers, but CDNs upstread typically strip out
// any extra headers from CONNECTs. As a result, the incoming requests often need to happen over HTTP (wrapped, of
// course, in whatever transport this server uses), and are upgraded here to HTTPS.
package httpsrewriter

import (
	"net/http"

	"github.com/getlantern/golog"
	"github.com/getlantern/http-proxy-lantern/domains"
	"github.com/getlantern/proxy/filters"
)

type rewriter struct {
	log golog.Logger
}

// NewRewriter creates a new request filter for rewriting requests to HTTPS services..
func NewRewriter() filters.Filter {
	return &rewriter{
		log: golog.LoggerFor("httpsRewritter"),
	}
}

// Apply implements the filters.Filter interface for HTTP request processing.
func (r *rewriter) Apply(ctx filters.Context, req *http.Request, next filters.Next) (*http.Response, filters.Context, error) {
	if req.Method == "CONNECT" {
		return next(ctx, req)
	}
	if cfg := domains.ConfigForRequest(req); cfg.RewriteToHTTPS {
		r.rewrite(cfg.Host, req)
	}
	return next(ctx, req)
}

func (r *rewriter) rewrite(host string, req *http.Request) {
	req.Host = host + ":443"
	req.URL.Host = req.Host
	req.URL.Scheme = "https"
	req.Close = false

	// The request URI is populated in the request to the proxy but raises an error if populated in outgoing client
	// requests.
	req.RequestURI = ""
	r.log.Debugf("Rewrote request with URL %#v to HTTPS", req.URL)
}
