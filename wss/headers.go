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
	}
)

type middleware struct{}

func NewMiddleware() *middleware {
	return &middleware{}
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
	if !(cfg.AddConfigServerHeaders) {
		return
	}

	conn := ctx.DownstreamConn()
	netx.WalkWrapped(conn, func(conn net.Conn) bool {
		switch t := conn.(type) {
		case *tinywss.WsConn:
			upHdr := t.UpgradeHeaders()
			// XXX use an auth token here to prove it's a CDN
			for _, header := range headerWhitelist {
				if val := upHdr.Get(header); val != "" {
					req.Header.Set(header, val)
					log.Debugf("WSS: copied header %s (%s)", header, val)
				} else {
					log.Debugf("WSS: header %s was not present!", header)
				}
			}
			return false
		}

		// Keep looking
		return true
	})
}
