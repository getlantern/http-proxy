package bbr

import (
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/getlantern/bbrconn"
	"github.com/getlantern/ema"
	"github.com/getlantern/golog"
	"github.com/getlantern/http-proxy-lantern/common"
	"github.com/getlantern/http-proxy/filters"
	"github.com/golang/groupcache/lru"
)

const (
	// don't record BBR info unless we've transferred at least 1 MB worth of data
	recordThreshold = 1024768

	// TODO: it probably makes sense to record BBR info at a few defined points,
	// like 10 KB, 1 MB, 10 MB
)

var (
	log = golog.LoggerFor("bbrlistener")
)

type stat struct {
	minRTT       *ema.EMA
	estBandwidth *ema.EMA
}

type Filter interface {
	filters.Filter
	Wrap(net.Listener) net.Listener
}

type bbrMiddleware struct {
	stats *lru.Cache
	mx    sync.RWMutex
}

func New() Filter {
	return &bbrMiddleware{
		stats: lru.New(5000),
	}
}

func (bm *bbrMiddleware) Apply(w http.ResponseWriter, req *http.Request, next filters.Next) error {
	remoteHost, _, _ := net.SplitHostPort(req.RemoteAddr)
	bm.mx.RLock()
	_stat, statFound := bm.stats.Get(remoteHost)
	bm.mx.RUnlock()
	if statFound {
		stat := _stat.(*stat)
		w.Header().Set(common.MinRTTHeader, stat.minRTT.GetDuration().String())
		w.Header().Set(common.EstBandwidthHeader, fmt.Sprint(stat.estBandwidth.Get()))
	}
	return next()
}

func (bm *bbrMiddleware) Wrap(l net.Listener) net.Listener {
	return &bbrlistener{l, bm}
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
	bbr, err := bbrconn.Wrap(conn)
	if err != nil {
		return nil, err
	}
	return &wrappedConn{Conn: bbr, bm: l.bm}, nil
}

func (l *bbrlistener) Addr() net.Addr {
	return l.wrapped.Addr()
}

func (l *bbrlistener) Close() error {
	return l.wrapped.Close()
}

type wrappedConn struct {
	bbrconn.Conn
	bm        *bbrMiddleware
	bytesSent uint64
}

func (c *wrappedConn) Write(b []byte) (int, error) {
	n, err := c.Conn.Write(b)
	c.bytesSent += uint64(n)
	return n, err
}

func (c *wrappedConn) Close() error {
	info, err := c.Info()
	if err != nil {
		log.Errorf("Unable to get bbr info: %v", err)
	} else {
		remoteHost, _, _ := net.SplitHostPort(c.RemoteAddr().String())
		log.Debugf("Estimated bandwidth to %v after sending %v: %v/s", remoteHost, humanize.Bytes(c.bytesSent), humanize.Bytes(uint64(info.EstBandwidth)))
		if c.bytesSent > recordThreshold {
			nextRTT := time.Duration(info.MinRTT)
			nextEstBandwidth := float64(info.EstBandwidth) * 8 / 1000 / 1000
			c.bm.mx.Lock()
			_stat, found := c.bm.stats.Get(remoteHost)
			if !found {
				stat := &stat{
					minRTT:       ema.NewDuration(nextRTT*time.Microsecond, 0.5),
					estBandwidth: ema.New(nextEstBandwidth, 0.5),
				}
				c.bm.stats.Add(remoteHost, stat)
				c.bm.mx.Unlock()
			} else {
				c.bm.mx.Unlock()
				stat := _stat.(*stat)
				stat.minRTT.UpdateDuration(nextRTT)
				stat.estBandwidth.Update(nextEstBandwidth)
			}
		}
	}
	return c.Conn.Close()
}
