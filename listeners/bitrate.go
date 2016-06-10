package listeners

import (
	"net"
	"net/http"

	"github.com/getlantern/http-proxy/listeners"
	"github.com/mxk/go-flowrate/flowrate"
)

type bitrateListener struct {
	net.Listener
	limit uint64
}

func NewBitrateListener(l net.Listener, lim uint64) net.Listener {
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
		proUser:            false,
		freader:            flowrate.NewReader(c, int64(bl.limit)),
		fwriter:            flowrate.NewWriter(c, int64(bl.limit)),
	}, err
}

// Bitrate Conn wrapper
type bitrateConn struct {
	listeners.WrapConnEmbeddable
	net.Conn
	active  bool
	proUser bool
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
	// pro-user message always overrides the active flag
	if msgType == "bitrate" && !c.proUser {
		log.Trace("Bitrate throttling message received")
		c.active = true
	} else if msgType == "pro-user" {
		c.proUser = true
		c.active = false
	}

	if c.WrapConnEmbeddable != nil {
		c.WrapConnEmbeddable.ControlMessage(msgType, data)
	}
}
