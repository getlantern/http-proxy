// Package http2direct short circuits the normal proxy processing of requests to make direct
// HTTP/2 connections to specific domains such as Lantern internal servers. This saves
// a significant amount of CPU in reducing TLS client handshakes but also makes these
// requests more efficient through the use of persistent connections and HTTP/2's multiplexing
// and reduced headers.
package http2direct

import (
	"net/http"
	"time"

	"github.com/getlantern/golog"
	"github.com/getlantern/http-proxy-lantern/domains"
	"github.com/getlantern/proxy/filters"
)

type http2direct struct {
	httpClient *http.Client
	log        golog.Logger
	traceID    uint64
}

// NewHTTP2Direct creates a new request filter for rewriting requests to HTTPS services..
func NewHTTP2Direct() filters.Filter {
	return &http2direct{
		httpClient: &http.Client{
			Transport: &http.Transport{
				IdleConnTimeout: 4 * time.Minute,
			},
		},
		log: golog.LoggerFor("httpsRewritter"),
	}
}

// Apply implements the filters.Filter interface for HTTP request processing.
func (h *http2direct) Apply(ctx filters.Context, req *http.Request, next filters.Next) (*http.Response, filters.Context, error) {
	if req.Method == "CONNECT" {
		return next(ctx, req)
	}
	if cfg := domains.ConfigForRequest(req); cfg.HTTP2Direct {
		// Make sure the request stays open.
		req.Close = false

		// The request URI is populated in the request to the proxy but raises an error if populated in outgoing client
		// requests.
		req.RequestURI = ""

		res, err := h.httpClient.Do(req)
		if err != nil {
			h.log.Errorf("Error short circuiting with HTTP/2 with req %#v, %v", req, err)
		}
		return res, ctx, nil
	}
	return next(ctx, req)
}
