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

	maxPendingHandshakesPerClient = 128
	maxHandshakesPerClient        = 4
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
		wrapped:  wrapped,
		sf:       sf,
		newConns: make(map[string]chan net.Conn),
		ready:    make(chan *result),
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
	newConns    map[string]chan net.Conn
	ready       chan *result
	handshaking int64
}

func (l *obfs4listener) Accept() (net.Conn, error) {
	r, ok := <-l.ready
	if !ok {
		return nil, fmt.Errorf("Closed")
	}
	return r.conn, r.err
}

func (l *obfs4listener) Addr() net.Addr {
	return l.wrapped.Addr()
}

func (l *obfs4listener) Close() error {
	err := l.wrapped.Close()
	go func() {
		// Drain ready
		for result := range l.ready {
			if result.conn != nil {
				result.conn.Close()
			}
		}
	}()
	return err
}

func (l *obfs4listener) accept() {
	defer func() {
		for _, newConns := range l.newConns {
			close(newConns)
		}
		close(l.ready)
	}()

	for {
		conn, err := l.wrapped.Accept()
		if err != nil {
			l.ready <- &result{nil, err}
			return
		}
		// WrapConn does a handshake with the client, which involves io operations
		// and can time out. We do it on a separate goroutine, but we limit it to
		// one goroutine per remote address.
		remoteAddr := conn.RemoteAddr().String()
		remoteHost, _, err := net.SplitHostPort(remoteAddr)
		if err != nil {
			log.Errorf("Unable to determine host for address %v: %v", remoteAddr, err)
			conn.Close()
			continue
		}
		newConns := l.newConns[remoteHost]
		if newConns == nil {
			newConns = make(chan net.Conn, maxPendingHandshakesPerClient)
			l.newConns[remoteHost] = newConns
			for i := 0; i < maxHandshakesPerClient; i++ {
				go l.wrap(newConns)
			}
		}
		select {
		case newConns <- conn:
			// will handshake
		default:
			log.Errorf("Too many pending handshakes for client at %v, ignoring new connections", remoteAddr)
			conn.Close()
		}
	}
}

func (l *obfs4listener) wrap(newConns chan net.Conn) {
	for conn := range newConns {
		l.doWrap(conn)
	}
}

func (l *obfs4listener) doWrap(conn net.Conn) {
	atomic.AddInt64(&l.handshaking, 1)
	defer atomic.AddInt64(&l.handshaking, -1)
	start := time.Now()
	_wrapped, timedOut, err := withtimeout.Do(handshakeTimeout, func() (interface{}, error) {
		o, err := l.sf.WrapConn(conn)
		if err != nil {
			return nil, err
		}
		return &obfs4Conn{Conn: o, wrapped: conn}, nil
	})

	if timedOut {
		log.Debugf("Handshake with %v timed out", conn.RemoteAddr())
		conn.Close()
	} else if err != nil {
		log.Debugf("Handshake error with %v: %v", conn.RemoteAddr(), err)
		conn.Close()
	} else {
		log.Debugf("Successful obfs4 handshake in %v", time.Now().Sub(start))
		l.ready <- &result{_wrapped.(net.Conn), err}
	}
}

func (l *obfs4listener) monitor() {
	for {
		time.Sleep(5 * time.Second)
		log.Debugf("Currently handshaking connections: %d", atomic.LoadInt64(&l.handshaking))
	}
}

type obfs4Conn struct {
	net.Conn
	wrapped net.Conn
}

func (conn *obfs4Conn) Wrapped() net.Conn {
	return conn.wrapped
}
