package listeners

import (
	"bufio"
	"bytes"
	"io"
	"net"
	"net/http"
	"strings"
	"sync/atomic"

	"github.com/getlantern/golog"

	"github.com/getlantern/http-proxy/listeners"

	"github.com/getlantern/http-proxy-lantern/mimic"
)

var log = golog.LoggerFor("listeners")

type preprocessorListener struct {
	net.Listener
}

func NewPreprocessorListener(l net.Listener) net.Listener {
	return &preprocessorListener{
		Listener: l,
	}
}

func (sl *preprocessorListener) Accept() (net.Conn, error) {
	c, err := sl.Listener.Accept()
	if err != nil {
		return nil, err
	}

	sac, _ := c.(listeners.WrapConnEmbeddable)
	return &preprocessorConn{
		WrapConnEmbeddable: sac,
		Conn:               c,
		newRequest:         1,
	}, err
}

// Preprocessor Conn wrapper
type preprocessorConn struct {
	listeners.WrapConnEmbeddable
	net.Conn
	// ready to handle a new http request when == 1
	newRequest uint32
}

func (c *preprocessorConn) Read(p []byte) (n int, err error) {
	if atomic.SwapUint32(&c.newRequest, 0) == 0 {
		return c.Conn.Read(p)
	}
	// TODO: user sync.Pool to avoid allocating memory for each request
	var buf bytes.Buffer
	r := io.TeeReader(c.Conn, &buf)
	n, err = r.Read(p)
	if err != nil {
		return
	}
	// On network with extremely small packet size, http header can be
	// fragmented to multiple IP packets, then a single Read() may not be able
	// to read the full http header, and ReadRequest() may fail. That would be
	// very rare.
	// It may also happen for purposely built clients, pipelined requests and
	// HTTP2 multiplexing. We know for sure that Lantern client will not issue
	// such requests, for now.
	req, e := http.ReadRequest(bufio.NewReader(&buf))
	defer func() {
		if req != nil && req.Body != nil {
			req.Body.Close()
		}
	}()
	if e != nil {
		// do nothing for network errors. ref (c *conn) serve() in net/http/server.go
		if e == io.EOF {
		} else if neterr, ok := e.(net.Error); ok && neterr.Timeout() {
		} else if e.Error() == "unexpected EOF" {
			// It can be an indicator that the request doesn't get sent in single IP packet.
			// Ignore it to avoid causing problem in such case.
		} else {
			log.Debugf("Error parse request from %s: %s", c.RemoteAddr().String(), e)
			// We have no way but check the error text
			if e.Error() == "parse : empty url" || strings.HasPrefix(e.Error(), "malformed HTTP ") {
				mimic.MimicApacheOnInvalidRequest(c.Conn, false)
			} else {
				mimic.MimicApacheOnInvalidRequest(c.Conn, true)
			}
			return 0, e
		}
	}
	return
}

func (c *preprocessorConn) OnState(s http.ConnState) {
	if s == http.StateIdle {
		atomic.StoreUint32(&c.newRequest, 1)
	}

	// Pass down to wrapped connections
	if c.WrapConnEmbeddable != nil {
		c.WrapConnEmbeddable.OnState(s)
	}
}

func (c *preprocessorConn) ControlMessage(msgType string, data interface{}) {
	// Simply pass down the control message to the wrapped connection
	if c.WrapConnEmbeddable != nil {
		c.WrapConnEmbeddable.ControlMessage(msgType, data)
	}
}
