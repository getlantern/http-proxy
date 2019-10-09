package http2direct

import (
	"net/http"
	"sync/atomic"
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
		log:     golog.LoggerFor("httpsRewritter"),
		traceID: 0,
	}
}

// Apply implements the filters.Filter interface for HTTP request processing.
func (h *http2direct) Apply(ctx filters.Context, req *http.Request, next filters.Next) (*http.Response, filters.Context, error) {
	if req.Method == "CONNECT" {
		return next(ctx, req)
	}
	if cfg := domains.ConfigForRequest(req); cfg.HTTP2Direct {
		defer atomic.AddUint64(&h.traceID, 1)
		res, err := h.httpClient.Do(req)
		if err != nil {
			h.log.Debugf("Error fetching with trace ID %v with req %#v, %v", h.traceID, req, err)
			return next(ctx, req)
		}
		h.log.Debugf("Response with trace ID %v: %#v", h.traceID, res)
		return res, ctx, nil
	}
	return next(ctx, req)
}
