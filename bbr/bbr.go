package bbr

import (
	"fmt"
	"net"
	"net/http"
	"sync"

	"github.com/dustin/go-humanize"
	"github.com/getlantern/bbrconn"
	"github.com/getlantern/golog"
	"github.com/getlantern/http-proxy-lantern/common"
	"github.com/getlantern/http-proxy/filters"
	"github.com/getlantern/netx"
	"github.com/getlantern/tcpinfo"
	"github.com/gorilla/context"
)

var (
	log = golog.LoggerFor("bbrlistener")
)

type Middleware struct {
	statsByClient map[string]*stats
	mx            sync.Mutex
}

func New() *Middleware {
	return &Middleware{
		statsByClient: make(map[string]*stats),
	}
}

// Apply implements the interface filters.Filter.
func (bm *Middleware) Apply(w http.ResponseWriter, req *http.Request, next filters.Next) error {
	bm.addMetrics(req, w.Header())
	return next()
}

func (bm *Middleware) AddMetrics(resp *http.Response) *http.Response {
	bm.addMetrics(resp.Request, resp.Header)
	return resp
}

func (bm *Middleware) addMetrics(req *http.Request, header http.Header) {
	if req.Header.Get(common.BBRRequested) == "" {
		// BBR info not requested, ignore
		return
	}
	_conn := context.Get(req, "conn")
	if _conn == nil {
		// TODO: for some reason, conn is nil when proxying HTTP requests. Figure
		// out why
		return
	}
	conn := _conn.(net.Conn)
	s := bm.statsFor(conn)
	netx.WalkWrapped(conn, func(conn net.Conn) bool {
		switch t := conn.(type) {
		case bbrconn.Conn:
			// Found bbr conn, get info
			bytesSent, info, infoErr := t.Info()
			bm.track(s, bytesSent, info, infoErr)
			return false
		}

		// Keep looking
		return true
	})
	header.Set(common.BBRAvailableBandwidthEstimateHeader, fmt.Sprint(s.estABE()))
}

func (bm *Middleware) statsFor(conn net.Conn) *stats {
	addr := conn.RemoteAddr().String()
	host, _, _ := net.SplitHostPort(addr)
	bm.mx.Lock()
	s := bm.statsByClient[host]
	if s == nil {
		s = newStats()
		bm.statsByClient[host] = s
	}
	bm.mx.Unlock()
	return s
}

func (bm *Middleware) track(s *stats, bytesSent int, info *tcpinfo.BBRInfo, err error) {
	if err != nil {
		log.Debugf("Unable to get BBR info (this happens when connections are closed unexpectedly): %v", err)
		return
	}
	s.update(float64(bytesSent), float64(info.EstBandwidth)*8/1000/1000)
	log.Debugf("Bytes sent: %v   BBR-ABE: %v   EMA BBR-ABE: %v", humanize.Bytes(uint64(bytesSent)), float64(info.EstBandwidth)*8/1000/1000, s.estABE())
}

func (bm *Middleware) Wrap(l net.Listener) net.Listener {
	log.Debugf("Enabling bbr metrics on %v", l.Addr())
	return &bbrlistener{l, bm}
}

type bbrlistener struct {
	net.Listener
	bm *Middleware
}

func (l *bbrlistener) Accept() (net.Conn, error) {
	conn, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}
	return bbrconn.Wrap(conn, func(bytesSent int, info *tcpinfo.BBRInfo, err error) {
		l.bm.track(l.bm.statsFor(conn), bytesSent, info, err)
	})
}
