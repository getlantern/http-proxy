package bbr

import (
	"fmt"
	"net"
	"net/http"

	"github.com/getlantern/bbrconn"
	"github.com/getlantern/golog"
	"github.com/getlantern/http-proxy-lantern/common"
	"github.com/getlantern/http-proxy/filters"
	"github.com/getlantern/netx"
	"github.com/gorilla/context"
)

var (
	log = golog.LoggerFor("bbrlistener")
)

type bbrMiddleware struct {
}

func NewFilter() filters.Filter {
	return &bbrMiddleware{}
}

// Apply implements the interface filters.Filter.
func (bm *bbrMiddleware) Apply(w http.ResponseWriter, req *http.Request, next filters.Next) error {
	addMetrics(req, w.Header())
	return next()
}

func Wrap(l net.Listener) net.Listener {
	log.Debugf("Enabling bbr metrics on %v", l.Addr())
	return &bbrlistener{l}
}

func AddMetrics(resp *http.Response) *http.Response {
	addMetrics(resp.Request, resp.Header)
	return resp
}

func addMetrics(req *http.Request, header http.Header) {
	_conn := context.Get(req, "conn")
	if _conn == nil {
		// TODO: for some reason, conn is nil when proxying HTTP requests. Figure
		// out why
		return
	}
	conn := _conn.(net.Conn)
	netx.WalkWrapped(conn, func(conn net.Conn) bool {
		switch t := conn.(type) {
		case bbrconn.Conn:
			// Found bbr conn, get info
			bytesSent, info, infoErr := t.Info()
			if infoErr != nil {
				log.Debugf("Unable to get BBR info (this happens when connections are closed unexpectedly): %v", infoErr)
				return false
			}
			bs := fmt.Sprint(bytesSent)
			abe := fmt.Sprint(float64(info.EstBandwidth) * 8 / 1000 / 1000)
			header.Set(common.BBRBytesSentHeader, bs)
			header.Set(common.BBRAvailableBandwidthEstimateHeader, abe)
			return false
		}

		// Keep looking
		return true
	})
}

type bbrlistener struct {
	net.Listener
}

func (l *bbrlistener) Accept() (net.Conn, error) {
	conn, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}
	return bbrconn.Wrap(conn)
}
