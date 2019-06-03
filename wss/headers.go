package wss

import (
	"net"
	"net/http"

	"github.com/getlantern/golog"
	"github.com/getlantern/http-proxy-lantern/domains"
	"github.com/getlantern/netx"
	"github.com/getlantern/proxy/filters"
	"github.com/getlantern/tinywss"
)

var (
	log = golog.LoggerFor("wss")
	// these headers are replicated from the inital http upgrade request
	// to certain subrequests on a wss connection.
	headerWhitelist = []string{
		"CloudFront-Viewer-Country",
		"X-Forwarded-For",
	}
)

const (
	CDNAuthTokenHeader = "X-Lantern-CDN-Auth-Token"
)

type middleware struct {
	authToken string
}

func NewMiddleware(authToken string) *middleware {
	return &middleware{authToken}
}

func (m *middleware) Apply(ctx filters.Context, req *http.Request, next filters.Next) (*http.Response, filters.Context, error) {
	m.apply(ctx, req)
	return next(ctx, req)
}

func (m *middleware) apply(ctx filters.Context, req *http.Request) {

	// carries through certain headers on authorized connections from CDNs
	// for domains that are configured to receive client ip information.
	// the connecting ip is a CDN edge server, so the req.RemoteAddr does
	// not reflect the correct client ip.

	cfg := domains.ConfigForRequest(req)
	if !(cfg.AddConfigServerHeaders || cfg.AddForwardedFor) {
		return
	}

	conn := ctx.DownstreamConn()
	netx.WalkWrapped(conn, func(conn net.Conn) bool {
		switch t := conn.(type) {
		case *tinywss.WsConn:
			upHdr := t.UpgradeHeaders()
			auth := upHdr.Get(CDNAuthTokenHeader)
			// this is an authorized CDN request, so we can trust these headers.
			if auth == m.authToken {
				for _, header := range headerWhitelist {
					if val := upHdr.Get(header); val != "" {
						req.Header.Set(header, val)
						log.Debugf("WSS: copied header %s (%s)", header, val)
					}
				}
			} else {
				log.Errorf("internal WSS request did not contain valid authorization header (%s='%s')", CDNAuthTokenHeader, auth)
			}
			return false
		}

		// Keep looking
		return true
	})
}
