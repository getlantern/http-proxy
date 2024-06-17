package water

import (
	"context"
	"io"
	"net"
	"sync"

	"github.com/getlantern/golog"
	"github.com/refraction-networking/water"
	_ "github.com/refraction-networking/water/transport/v0"
)

var log = golog.LoggerFor("water")

type llistener struct {
	net.Listener

	connections  chan net.Conn
	closedSignal chan struct{}
	closeOnce    sync.Once
	closeError   error
}

func NewReverseListener(ctx context.Context, address string, wasm []byte) (*llistener, error) {
	cfg := &water.Config{
		TransportModuleBin: wasm,
	}

	waterListener, err := cfg.ListenContext(ctx, "tcp", address)
	if err != nil {
		log.Errorf("error creating water listener: %v", err)
		return nil, err
	}

	l := &innerListener{Listener: waterListener}
	ll := &llistener{
		Listener:     l,
		connections:  make(chan net.Conn),
		closedSignal: make(chan struct{}),
	}

	accept := func() (net.Conn, error) {
		return ll.Listener.Accept()
	}

	// the handler must receive the connection and send it to the l.connections channel
	handler := func(conn net.Conn) {
		handleReverseConnection(conn, ll.connections)
	}

	go serve(accept, handler)
	return ll, nil
}

func (l *llistener) Accept() (net.Conn, error) {
	select {
	case c := <-l.connections:
		return c, nil
	case <-l.closedSignal:
		return nil, l.closeError
	}
}

func (l *llistener) Close() error {
	l.closeOnce.Do(func() {
		close(l.closedSignal)
		l.closeError = l.Close()
	})
	return l.closeError
}

type acceptWATER func() (net.Conn, error)
type handleWATER func(net.Conn)

func serve(accept acceptWATER, handle handleWATER) {
	for {
		conn, err := accept()
		if err != nil {
			log.Errorf("accept: %v", err)
			return
		}
		go handle(conn)
	}
}

func handleReverseConnection(conn net.Conn, connections chan net.Conn) {
	log.Debugf("handling connection from/to %s", conn.RemoteAddr())

	c1, c2 := net.Pipe()
	a := c1
	b := &lfwd{
		Conn:           c2,
		clientTCPConn:  conn,
		remoteAddr:     conn.RemoteAddr(),
		upstreamTarget: conn.RemoteAddr().String(),
	}
	connections <- b

	fromClientErrCh := make(chan error)
	go func() {
		defer a.Close()
		defer conn.Close()
		_, err := io.Copy(a, conn)
		if err != nil {
			io.Copy(io.Discard, conn)
		}
		fromClientErrCh <- err
	}()

	_, fromTargetErr := io.Copy(conn, a)
	fromClientErr := <-fromClientErrCh
	if fromClientErr != nil {
		log.Errorf("failed to relay traffic from client: %v", fromClientErr)
	}
	if fromTargetErr != nil {
		log.Errorf("failed to relay traffic from target: %v", fromTargetErr)
	}
}

type innerListener struct {
	net.Listener
}

func (il *innerListener) Accept() (net.Conn, error) {
	c, err := il.Listener.Accept()
	if err != nil {
		return nil, err
	}

	return c, nil
}

// this is an adapter that forwards the remote address
// on the "real" client connection to the consumer of
// the listener.  The real requested upstream address
// is also available if needed.
type lfwd struct {
	net.Conn
	clientTCPConn  net.Conn
	remoteAddr     net.Addr
	upstreamTarget string
}

func (l *lfwd) RemoteAddr() net.Addr {
	return l.remoteAddr
}

func (l *lfwd) UpstreamTarget() string {
	return l.upstreamTarget
}

func (l *lfwd) Wrapped() net.Conn {
	return l.clientTCPConn
}
