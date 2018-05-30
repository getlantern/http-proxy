package listeners

import (
	"net"
	"net/http"
	"sync/atomic"

	"github.com/mxk/go-flowrate/flowrate"

	"github.com/getlantern/http-proxy/listeners"
)

type RateLimiter struct {
	rm   *flowrate.Monitor
	wm   *flowrate.Monitor
	rate int64
}

func NewRateLimiter(rate int64) *RateLimiter {
	return &RateLimiter{flowrate.New(0, 0), flowrate.New(0, 0), rate}
}

func (l *RateLimiter) SetRate(rate int64) {
	atomic.StoreInt64(&l.rate, rate)
}

func (l *RateLimiter) getRate() int64 {
	return atomic.LoadInt64(&l.rate)
}

type bitrateListener struct {
	net.Listener
}

func NewBitrateListener(l net.Listener) net.Listener {
	return &bitrateListener{l}
}

func (bl *bitrateListener) Accept() (net.Conn, error) {
	c, err := bl.Listener.Accept()
	if err != nil {
		return nil, err
	}

	wc, _ := c.(listeners.WrapConnEmbeddable)
	return &bitrateConn{
		WrapConnEmbeddable: wc,
		Conn:               c,
		limiter:            NewRateLimiter(0),
	}, err
}

// Bitrate Conn wrapper
type bitrateConn struct {
	listeners.WrapConnEmbeddable
	net.Conn
	limiter *RateLimiter
}

func (c *bitrateConn) Read(p []byte) (n int, err error) {
	rate := c.limiter.getRate()
	if rate == 0 {
		return c.Conn.Read(p)
	}
	s := c.limiter.rm.Limit(len(p), rate, true)
	if s > 0 {
		n, err = c.limiter.rm.IO(c.Conn.Read(p[:s]))
	}
	return
}

func (c *bitrateConn) Write(p []byte) (n int, err error) {
	rate := c.limiter.getRate()
	if rate == 0 {
		return c.Conn.Write(p)
	}
	var i int
	for len(p) > 0 && err == nil {
		s := c.limiter.wm.Limit(len(p), rate, true)
		if s > 0 {
			i, err = c.limiter.wm.IO(c.Conn.Write(p[:s]))
		} else {
			return n, flowrate.ErrLimit
		}
		p = p[i:]
		n += i
	}
	return
}

func (c *bitrateConn) OnState(s http.ConnState) {
	// Pass down to wrapped connections
	if c.WrapConnEmbeddable != nil {
		c.WrapConnEmbeddable.OnState(s)
	}
}

func (c *bitrateConn) ControlMessage(msgType string, data interface{}) {
	// pro-user message always overrides the active flag
	if msgType == "throttle" {
		c.limiter = data.(*RateLimiter)
	}

	if c.WrapConnEmbeddable != nil {
		c.WrapConnEmbeddable.ControlMessage(msgType, data)
	}
}

func (c *bitrateConn) Wrapped() net.Conn {
	return c.Conn
}
