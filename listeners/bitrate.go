package listeners

import (
	"net"
	"net/http"

	"github.com/dustin/go-humanize"
	"github.com/getlantern/http-proxy/listeners"
	"github.com/mxk/go-flowrate/flowrate"
)

type ThrottleRate int64

var (
	NoThrottle = ThrottleRate(0)
)

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
		rm:                 flowrate.New(0, 0),
		wm:                 flowrate.New(0, 0),
		throttle:           0,
	}, err
}

// Bitrate Conn wrapper
type bitrateConn struct {
	listeners.WrapConnEmbeddable
	net.Conn
	rm       *flowrate.Monitor
	wm       *flowrate.Monitor
	throttle int64
}

func (c *bitrateConn) Read(p []byte) (n int, err error) {
	if c.throttle == 0 {
		return c.Conn.Read(p)
	}
	s := c.rm.Limit(len(p), c.throttle, true)
	if s > 0 {
		n, err = c.rm.IO(c.Conn.Read(p[:s]))
	}
	return
}

func (c *bitrateConn) Write(p []byte) (n int, err error) {
	if c.throttle == 0 {
		return c.Conn.Write(p)
	}
	var i int
	for len(p) > 0 && err == nil {
		s := c.wm.Limit(len(p), c.throttle, true)
		if s > 0 {
			i, err = c.wm.IO(c.Conn.Write(p[:s]))
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
		rate := data.(ThrottleRate)
		c.throttle = int64(rate)
		log.Debugf("Throttling connection to %v per second", humanize.Bytes(uint64(rate)))
	}

	if c.WrapConnEmbeddable != nil {
		c.WrapConnEmbeddable.ControlMessage(msgType, data)
	}
}

func (c *bitrateConn) Wrapped() net.Conn {
	return c.Conn
}
