package bbr

import (
	"net"

	"github.com/dustin/go-humanize"
	"github.com/getlantern/bbrconn"
	"github.com/getlantern/golog"
)

var (
	log = golog.LoggerFor("bbrlistener")
)

type bbrlistener struct {
	wrapped net.Listener
}

func Wrap(l net.Listener) net.Listener {
	return &bbrlistener{l}
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
	return &wrappedConn{Conn: bbr}, nil
}

func (l *bbrlistener) Addr() net.Addr {
	return l.wrapped.Addr()
}

func (l *bbrlistener) Close() error {
	return l.wrapped.Close()
}

type wrappedConn struct {
	bbrconn.Conn
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
		log.Debugf("Estimated bandwidth to %v after sending %v: %v/s", c.RemoteAddr(), humanize.Bytes(c.bytesSent), humanize.Bytes(uint64(info.EstBandwidth)))
	}
	return c.Conn.Close()
}
