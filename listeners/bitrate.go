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
		throttle:           NoThrottle,
		freader:            flowrate.NewReader(c, int64(0)),
		fwriter:            flowrate.NewWriter(c, int64(0)),
	}, err
}

// Bitrate Conn wrapper
type bitrateConn struct {
	listeners.WrapConnEmbeddable
	net.Conn
	throttle ThrottleRate
	freader  *flowrate.Reader
	fwriter  *flowrate.Writer
}

func (c *bitrateConn) Read(p []byte) (n int, err error) {
	if c.throttle == NoThrottle {
		return c.Conn.Read(p)
	} else {
		return c.freader.Read(p)
	}
}

func (c *bitrateConn) Write(p []byte) (n int, err error) {
	if c.throttle == NoThrottle {
		return c.Conn.Write(p)
	} else {
		return c.fwriter.Write(p)
	}
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
		c.throttle = rate
		c.freader.SetLimit(int64(c.throttle))
		c.fwriter.SetLimit(int64(c.throttle))
		log.Debugf("Throttling connection to %v per second", humanize.Bytes(uint64(rate)))
	}

	if c.WrapConnEmbeddable != nil {
		c.WrapConnEmbeddable.ControlMessage(msgType, data)
	}
}

func (c *bitrateConn) Wrapped() net.Conn {
	return c.Conn
}
