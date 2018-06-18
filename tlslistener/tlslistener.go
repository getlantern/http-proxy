// Package tlslistener provides a wrapper around tls.Listen that allows
// descending into the wrapped net.Conn
package tlslistener

import (
	"crypto/tls"
	"net"

	"github.com/getlantern/golog"
	"github.com/getlantern/tlsdefaults"
)

// Wrap wraps the specified listener in our default TLS listener.
func Wrap(wrapped net.Listener, keyFile string, certFile string) (net.Listener, error) {
	cfg, err := tlsdefaults.BuildListenerConfig(wrapped.Addr().String(), keyFile, certFile)
	if err != nil {
		return nil, err
	}

	listener := &tlslistener{wrapped, cfg, golog.LoggerFor("lantern-proxy-tlslistener")}
	cfg.GetConfigForClient = listener.debugClientHello
	return listener, nil
}

type tlslistener struct {
	wrapped net.Listener
	cfg     *tls.Config
	log     golog.Logger
}

func (l *tlslistener) Accept() (net.Conn, error) {
	conn, err := l.wrapped.Accept()
	if err != nil {
		return nil, err
	}
	return &tlsconn{tls.Server(conn, l.cfg), conn}, nil
}

func (l *tlslistener) Addr() net.Addr {
	return l.wrapped.Addr()
}

func (l *tlslistener) Close() error {
	return l.wrapped.Close()
}

func (l *tlslistener) debugClientHello(info *tls.ClientHelloInfo) (*tls.Config, error) {
	l.log.Debugf("Cipher suites from client %v for server suites %v "+info.Conn.RemoteAddr().String()+"->"+info.Conn.LocalAddr().String(), info.CipherSuites, l.cfg.CipherSuites)
	return nil, nil
}

type tlsconn struct {
	net.Conn
	wrapped net.Conn
}

func (conn *tlsconn) Wrapped() net.Conn {
	return conn.wrapped
}
