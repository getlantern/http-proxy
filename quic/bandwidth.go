package quic

import (
	"fmt"
	"net"
	"net/http"

	"github.com/getlantern/golog"
	"github.com/getlantern/http-proxy-lantern/v2/common"
	"github.com/getlantern/netx"
	"github.com/getlantern/proxy/filters"
	"github.com/getlantern/quicwrapper"
)

var (
	log = golog.LoggerFor("quic")
)

type middleware struct{}

func NewMiddleware() *middleware {
	return &middleware{}
}

func (m *middleware) Apply(ctx filters.Context, req *http.Request, next filters.Next) (*http.Response, filters.Context, error) {

	resp, nextCtx, err := next(ctx, req)
	if resp != nil {
		m.apply(ctx, req, resp)
	}
	return resp, nextCtx, err
}

func (m *middleware) apply(ctx filters.Context, req *http.Request, resp *http.Response) {
	// This gives back a BBR ABE response header when requested based on quic's
	// bandwidth estimate ... not actually BBR and without the particular averaging
	// done by the bbr middleware.
	conn := ctx.DownstreamConn()

	bbrRequested := req.Header.Get(common.BBRRequested)
	if bbrRequested == "" {
		log.Tracef("No BBR estimate requested...")
		// BBR info not requested, ignore
		return
	}

	var estABE float64
	netx.WalkWrapped(conn, func(conn net.Conn) bool {
		if t, ok := conn.(*quicwrapper.Conn); ok {
			estABE = float64(t.BandwidthEstimate()) / quicwrapper.Mib
			return false
		}
		// Keep looking
		return true
	})

	log.Tracef("Quic estABE = %v", estABE)
	if resp.Header == nil {
		resp.Header = make(http.Header, 1)
	}
	resp.Header.Set(common.BBRAvailableBandwidthEstimateHeader, fmt.Sprint(estABE))
}
