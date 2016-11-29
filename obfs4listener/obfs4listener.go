package obfs4listener

import (
	"fmt"
	"net"
	"os"
	"sync/atomic"
	"time"

	"github.com/getlantern/golog"
	"github.com/getlantern/withtimeout"

	"git.torproject.org/pluggable-transports/goptlib.git"
	"git.torproject.org/pluggable-transports/obfs4.git/transports/base"
	"git.torproject.org/pluggable-transports/obfs4.git/transports/obfs4"
)

var (
	log = golog.LoggerFor("obfs4listener")

	handshakeTimeout = 10 * time.Second
)

func Wrap(wrapped net.Listener, stateDir string) (net.Listener, error) {
	err := os.MkdirAll(stateDir, 0700)
	if err != nil {
		return nil, fmt.Errorf("Unable to make statedir at %v: %v", stateDir, err)
	}

	tr := &obfs4.Transport{}
	sf, err := tr.ServerFactory(stateDir, &pt.Args{})
	if err != nil {
		return nil, fmt.Errorf("Unable to create obfs4 server factory: %v", err)
	}

	ol := &obfs4listener{
		wrapped: wrapped,
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
	wrapped     net.Listener
	sf          base.ServerFactory
	ready       chan *result
	handshaking int64
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
	atomic.AddInt64(&l.handshaking, 1)
	_wrapped, timedOut, err := withtimeout.Do(handshakeTimeout, func() (interface{}, error) {
		return l.sf.WrapConn(conn)
	})
	atomic.AddInt64(&l.handshaking, -1)
	if timedOut {
		log.Debugf("Handshake with %v timed out", conn.RemoteAddr())
		conn.Close()
	} else if err != nil {
		log.Debugf("Handshake error with %v: %v", conn.RemoteAddr(), err)
		conn.Close()
	} else {
		l.ready <- &result{_wrapped.(net.Conn), err}
	}
}

func (l *obfs4listener) monitor() {
	for {
		time.Sleep(5 * time.Second)
		log.Debugf("Currently handshaking connections: %d", atomic.LoadInt64(&l.handshaking))
	}
}
