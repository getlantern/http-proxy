package listeners

import (
	"net"
	"net/http"

	"github.com/getlantern/http-proxy/listeners"
	"github.com/mxk/go-flowrate/flowrate"
)

type bitrateListener struct {
	net.Listener
	limit int64
}

func NewBitrateListener(l net.Listener, lim int64) net.Listener {
	return &bitrateListener{
		Listener: l,
		limit:    lim,
	}
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
		active:             false,
		freader:            flowrate.NewReader(c, bl.limit),
		fwriter:            flowrate.NewWriter(c, bl.limit),
	}, err
}

// Bitrate Conn wrapper
type bitrateConn struct {
	listeners.WrapConnEmbeddable
	net.Conn
	active  bool
	freader *flowrate.Reader
	fwriter *flowrate.Writer
}

func (c *bitrateConn) Read(p []byte) (n int, err error) {
	if c.active {
		return c.freader.Read(p)
	} else {
		return c.Conn.Read(p)
	}
}

func (c *bitrateConn) Write(p []byte) (n int, err error) {
	if c.active {
		return c.fwriter.Write(p)
	} else {
		return c.Conn.Write(p)
	}
}

func (c *bitrateConn) OnState(s http.ConnState) {
	// Pass down to wrapped connections
	if c.WrapConnEmbeddable != nil {
		c.WrapConnEmbeddable.OnState(s)
	}
}

func (c *bitrateConn) ControlMessage(msgType string, data interface{}) {
	if msgType == "bitrate" {
		log.Trace("Bitrate throttling message received")
		c.active = true
	}

	if c.WrapConnEmbeddable != nil {
		c.WrapConnEmbeddable.ControlMessage(msgType, data)
	}
}
