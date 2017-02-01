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

type Filter interface {
	filters.Filter
	Wrap(net.Listener) net.Listener
}

type bbrMiddleware struct {
}

func New() Filter {
	return &bbrMiddleware{}
}

// Apply implements the interface filters.Filter.
func (bm *bbrMiddleware) Apply(w http.ResponseWriter, req *http.Request, next filters.Next) error {
	bm.addMetrics(req, w.Header())
	return next()
}

func (bm *bbrMiddleware) Wrap(l net.Listener) net.Listener {
	return &bbrlistener{l, bm}
}

func (bm *bbrMiddleware) addMetrics(req *http.Request, header http.Header) {
	conn := context.Get(req, "conn").(net.Conn)
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
	wrapped net.Listener
	bm      *bbrMiddleware
}

func (l *bbrlistener) Accept() (net.Conn, error) {
	conn, err := l.wrapped.Accept()
	if err != nil {
		return nil, err
	}
	return bbrconn.Wrap(conn)
}

func (l *bbrlistener) Addr() net.Addr {
	return l.wrapped.Addr()
}

func (l *bbrlistener) Close() error {
	return l.wrapped.Close()
}
