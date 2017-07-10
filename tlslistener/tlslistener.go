// Package tlslistener provides a wrapper around tls.Listen that allows
// descending into the wrapped net.Conn
package tlslistener

import (
	"crypto/tls"
	"net"
	"reflect"

	"github.com/getlantern/golog"
	"github.com/getlantern/tlsdefaults"
)

var log = golog.LoggerFor("http-proxy.tlslistener")

func Wrap(wrapped net.Listener, keyFile string, certFile string) (net.Listener, error) {
	cfg, err := tlsdefaults.BuildListenerConfig(wrapped.Addr().String(), keyFile, certFile)
	if err != nil {
		return nil, err
	}
	return &tlslistener{wrapped, cfg}, nil
}

type tlslistener struct {
	wrapped net.Listener
	cfg     *tls.Config
}

func (l *tlslistener) Accept() (net.Conn, error) {
	log.Debugf("Accepting from underlying: %v", reflect.TypeOf(l.wrapped))
	conn, err := l.wrapped.Accept()
	if err != nil {
		return nil, err
	}
	log.Debugf("Building tls conn")
	result := &tlsconn{tls.Server(conn, l.cfg), conn}
	log.Debugf("Returning tls conn")
	return result, nil
}

func (l *tlslistener) Addr() net.Addr {
	return l.wrapped.Addr()
}

func (l *tlslistener) Close() error {
	return l.wrapped.Close()
}

type tlsconn struct {
	net.Conn
	wrapped net.Conn
}

func (conn *tlsconn) Wrapped() net.Conn {
	return conn.wrapped
}
