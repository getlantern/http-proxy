package obfs4listener

import (
	"fmt"
	"net"
	"os"
	"time"

	"github.com/Yawning/obfs4/transports/obfs4"
	"github.com/getlantern/golog"
	"github.com/getlantern/withtimeout"

	"git.torproject.org/pluggable-transports/goptlib.git"
	"git.torproject.org/pluggable-transports/obfs4.git/transports/base"
)

var (
	log = golog.LoggerFor("obfs4listener")

	handshakeTimeout = 30 * time.Second
)

func NewListener(addr string, stateDir string) (net.Listener, error) {
	err := os.MkdirAll(stateDir, 0700)
	if err != nil {
		return nil, fmt.Errorf("Unable to make statedir at %v: %v", stateDir, err)
	}

	tr := &obfs4.Transport{}
	sf, err := tr.ServerFactory(stateDir, &pt.Args{})
	if err != nil {
		return nil, fmt.Errorf("Unable to create obfs4 server factory: %v", err)
	}
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("Unable to listen at %v: %v", addr, err)
	}

	ol := &obfs4listener{
		wrapped: l,
		sf:      sf,
		ready:   make(chan *result, 1000),
	}

	go ol.accept()
	go ol.monitor()
	return ol, nil
}

type result struct {
	conn net.Conn
	err  error
}

type obfs4listener struct {
	wrapped net.Listener
	sf      base.ServerFactory
	ready   chan *result
}

func (l *obfs4listener) Accept() (net.Conn, error) {
	r := <-l.ready
	return r.conn, r.err
}

func (l *obfs4listener) Addr() net.Addr {
	return l.wrapped.Addr()
}

func (l *obfs4listener) Close() error {
	return l.wrapped.Close()
}

func (l *obfs4listener) accept() {
	for {
		conn, err := l.wrapped.Accept()
		if err != nil {
			l.ready <- &result{nil, err}
		} else {
			// WrapConn does a handshake with the client, which involves io operations
			// and can time out
			go l.wrap(conn)
		}
	}
}

func (l *obfs4listener) wrap(conn net.Conn) {
	_wrapped, timedOut, err := withtimeout.Do(handshakeTimeout, func() (interface{}, error) {
		return l.sf.WrapConn(conn)
	})
	if timedOut {
		log.Debugf("Handshake with %v timed out", conn.RemoteAddr())
	} else if err != nil {
		log.Debugf("Handshake error with %v: %v", conn.RemoteAddr(), err)
	} else {
		l.ready <- &result{_wrapped.(net.Conn), err}
	}
}

func (l *obfs4listener) monitor() {
	for {
		time.Sleep(5 * time.Second)
		log.Debugf("Currently handshaking connections: %d", len(l.ready))
	}
}
