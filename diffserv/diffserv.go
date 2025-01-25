// Package diffserv provides a listener that adds diffserv TOS (a.k.a. QOS)
// headers. For good descriptions of available TOS values, see
// https://www.tucny.com/Home/dscp-tos.
package diffserv

import (
	"net"

	"golang.org/x/net/ipv4"

	"github.com/getlantern/http-proxy-lantern/v2/logger"
)

var (
	log = logger.InitLogger("diffserv")
)

// Wrap wraps the given Listener into a Listener that applies the specified tos
// to all connections.
func Wrap(l net.Listener, tos int) net.Listener {
	return &diffservListener{l, tos}
}

type diffservListener struct {
	wrapped net.Listener
	tos     int
}

func (l *diffservListener) Accept() (net.Conn, error) {
	conn, err := l.wrapped.Accept()
	if err != nil {
		return conn, err
	}
	tosErr := ipv4.NewConn(conn).SetTOS(l.tos)
	if tosErr != nil {
		log.Errorf("Unable to set TOS to %d: %v", l.tos, tosErr)
	}
	return conn, nil
}

func (l *diffservListener) Addr() net.Addr {
	return l.wrapped.Addr()
}

func (l *diffservListener) Close() error {
	return l.wrapped.Close()
}
