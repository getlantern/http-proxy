package bbr

import (
	"fmt"
	"net"
	"net/http"
	"runtime"
	"sync"

	"github.com/getlantern/bbrconn"
	borda "github.com/getlantern/borda/client"
	"github.com/getlantern/golog"
	"github.com/getlantern/http-proxy-lantern/common"
	"github.com/getlantern/http-proxy/filters"
	"github.com/getlantern/netx"
	"github.com/getlantern/ops"
	"github.com/getlantern/tcpinfo"
	"github.com/gorilla/context"
)

type middleware struct {
	statsByClient map[string]*stats
	mx            sync.Mutex
}

func New() Middleware {
	if runtime.GOOS == "linux" {
		log.Debug("Tracking bbr metrics on Linux")
		return &middleware{
			statsByClient: make(map[string]*stats),
		}
	}
	log.Debugf("Not tracking bbr metrics on %v", runtime.GOOS)
	return &noopMiddleware{}
}

// Apply implements the interface filters.Filter.
func (bm *middleware) Apply(w http.ResponseWriter, req *http.Request, next filters.Next) error {
	bm.addMetrics(req, w.Header())
	return next()
}

func (bm *middleware) AddMetrics(resp *http.Response) *http.Response {
	bm.addMetrics(resp.Request, resp.Header)
	return resp
}

func (bm *middleware) addMetrics(req *http.Request, header http.Header) {
	conn := connFor(req)
	if conn == nil {
		// TODO: for some reason, conn is nil when proxying HTTP requests. Figure
		// out why
		return
	}
	s := bm.statsFor(conn)

	bbrRequested := req.Header.Get(common.BBRRequested)
	clear := bbrRequested == "clear"
	if clear {
		log.Debugf("Clearing stats for %v", conn.RemoteAddr())
		s.clear()
	}

	netx.WalkWrapped(conn, func(conn net.Conn) bool {
		switch t := conn.(type) {
		case bbrconn.Conn:
			// Found bbr conn, get info
			bytesSent := t.BytesWritten()
			bbrInfo, infoErr := t.BBRInfo()
			bm.track(false, s, conn.RemoteAddr(), bytesSent, nil, bbrInfo, infoErr)
			return false
		}

		// Keep looking
		return true
	})

	if bbrRequested == "" {
		// BBR info not requested, ignore
		return
	}
	header.Set(common.BBRAvailableBandwidthEstimateHeader, fmt.Sprint(s.estABE()))
}

func (bm *middleware) statsFor(conn net.Conn) *stats {
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

func (bm *middleware) track(reportToBorda bool, s *stats, remoteAddr net.Addr, bytesSent int, info *tcpinfo.Info, bbrInfo *tcpinfo.BBRInfo, err error) {
	if err != nil {
		log.Debugf("Unable to get BBR info (this happens when connections are closed unexpectedly): %v", err)
		return
	}
	s.update(float64(bytesSent), float64(bbrInfo.EstBandwidth)*8/1000/1000)
	if reportToBorda {
		go func() {
			// We do this inside a goroutine because we explicitly don't want to inherit
			// the existing context (to reduce data volumes to borda)
			op := ops.Begin("tcpinfo")
			clientIP, _, err := net.SplitHostPort(remoteAddr.String())
			if err == nil {
				op.Set("client_ip", clientIP)
			}
			op.Set("bytes_sent", borda.Sum(bytesSent))
			op.Set("tcp_sender_mss", borda.Avg(float64(info.SenderMSS)))
			op.Set("tcp_rtt", borda.Avg(float64(info.RTT/nanosPerMilli)))
			op.Set("tcp_segments_sent", borda.Sum(float64(info.Sys.SegsOut)))
			op.Set("tcp_segments_sent_retransmitted", borda.Sum(float64(info.Sys.TotalRetransSegs)))
			estMbps := s.estABE()
			if estMbps > 0 {
				// Report ABE if available
				op.Set("est_mbps", borda.Avg(estMbps))
				op.Set("est_mbps_min", borda.Min(estMbps))
				op.Set("est_mbps_max", borda.Max(estMbps))
			}
			log.Debugf("reporting tcp info")
			op.End()
		}()
	}
}

func (bm *middleware) Wrap(l net.Listener) net.Listener {
	log.Debugf("Enabling bbr metrics on %v", l.Addr())
	return &bbrlistener{l, bm}
}

func (bm *middleware) ABE(req *http.Request) float64 {
	conn := connFor(req)
	if conn == nil {
		return 0
	}
	return bm.statsFor(conn).estABE()
}

type bbrlistener struct {
	net.Listener
	bm *middleware
}

func (l *bbrlistener) Accept() (net.Conn, error) {
	conn, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}
	return bbrconn.Wrap(conn, func(bytesSent int, info *tcpinfo.Info, bbrInfo *tcpinfo.BBRInfo, err error) {
		l.bm.track(true, l.bm.statsFor(conn), conn.RemoteAddr(), bytesSent, info, bbrInfo, err)
	})
}

type noopMiddleware struct{}

func (nm *noopMiddleware) Apply(w http.ResponseWriter, req *http.Request, next filters.Next) error {
	return next()
}

func (nm *noopMiddleware) AddMetrics(resp *http.Response) *http.Response {
	return resp
}

func (nm *noopMiddleware) Wrap(l net.Listener) net.Listener {
	return l
}

func (nm *noopMiddleware) ABE(req *http.Request) float64 {
	return 0
}

func connFor(req *http.Request) net.Conn {
	_conn := context.Get(req, "conn")
	if _conn == nil {
		return nil
	}
	return _conn.(net.Conn)
}
