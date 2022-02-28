package quic

import (
	"fmt"
	"net"
	"net/http"

	"github.com/getlantern/zaplog"
	"github.com/getlantern/http-proxy-lantern/v2/common"
	"github.com/getlantern/netx"
	"github.com/getlantern/proxy/v2/filters"
	"github.com/getlantern/quicwrapper"
)

var (
	log = zaplog.LoggerFor("quic")
)

type middleware struct{}

func NewMiddleware() *middleware {
	return &middleware{}
}

func (m *middleware) Apply(cs *filters.ConnectionState, req *http.Request, next filters.Next) (*http.Response, *filters.ConnectionState, error) {

	resp, nextCtx, err := next(cs, req)
	if resp != nil {
		m.apply(cs, req, resp)
	}
	return resp, nextCtx, err
}

func (m *middleware) apply(cs *filters.ConnectionState, req *http.Request, resp *http.Response) {
	// This gives back a BBR ABE response header when requested based on quic's
	// bandwidth estimate ... not actually BBR and without the particular averaging
	// done by the bbr middleware.
	conn := cs.Downstream()

	bbrRequested := req.Header.Get(common.BBRRequested)
	if bbrRequested == "" {
		log.Debugf("No BBR estimate requested...")
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

	log.Debugf("Quic estABE = %v", estABE)
	if resp.Header == nil {
		resp.Header = make(http.Header, 1)
	}
	resp.Header.Set(common.BBRAvailableBandwidthEstimateHeader, fmt.Sprint(estABE))
}
