package shadowsocks

import (
	"context"
	"errors"
	"net"
	"time"

	"github.com/getlantern/netx"

	"github.com/Jigsaw-Code/outline-sdk/transport"
	onet "github.com/Jigsaw-Code/outline-ss-server/net"
	"github.com/Jigsaw-Code/outline-ss-server/service"

	"github.com/getlantern/http-proxy-lantern/v2/logger"
)

// shadowsocks/local.go houses adapters for use with Lantern. This mostly is in
// place to allow the Lantern backend to handle upstream connections itself and
// have shadowsocks behave like other transports we use in Lantern.

var (
	log = logger.InitLogger("shadowsocks")

	ErrListenerClosed = errors.New("listener closed")
)

// This value is lifted from the the main server.go to match behavior
// 59 seconds is most common timeout for servers that do not respond to invalid requests
const tcpReadTimeout time.Duration = 59 * time.Second

// ListenLocalTCP creates a net.Listener that returns all inbound shadowsocks connections to the
// returned listener rather than dialing upstream. Any upstream or local handling should be handled by the
// caller of Accept().
func ListenLocalTCP(
	l net.Listener,
	ciphers service.CipherList,
	replayHistory int,
) (net.Listener, error) {
	replayCache := service.NewReplayCache(replayHistory)

	options := &ListenerOptions{
		Listener:           &tcpListenerAdapter{l},
		Ciphers:            ciphers,
		ReplayCache:        &replayCache,
		ShadowsocksMetrics: &service.NoOpTCPMetrics{},
	}

	return ListenLocalTCPOptions(options), nil
}

// ListenLocalTCPOptions creates a net.Listener that returns some inbound shadowsocks connections
// to the returned listener.  Which connnections are returned to the listener are determined
// by the ShouldHandleLocally predicate, which defaults to all connections.
// Any upstream handling should be handled by the caller of Accept() for any connection returned.
func ListenLocalTCPOptions(options *ListenerOptions) net.Listener {
	maxPending := options.MaxPendingConnections
	if maxPending == 0 {
		maxPending = DefaultMaxPending
	}

	l := &llistener{
		wrapped:      options.Listener,
		connections:  make(chan net.Conn, maxPending),
		closedSignal: make(chan struct{}),
	}

	timeout := options.Timeout
	if timeout == 0 {
		timeout = tcpReadTimeout
	}

	validator := options.TargetIPValidator
	if validator == nil {
		validator = onet.RequirePublicIP
	}

	authFunc := service.NewShadowsocksStreamAuthenticator(options.Ciphers, options.ReplayCache, options.ShadowsocksMetrics)
	tcpHandler := service.NewTCPHandler(options.Listener.Addr().(*net.TCPAddr).Port, authFunc, options.ShadowsocksMetrics, timeout)
	tcpHandler.SetTargetDialer(&LocalDialer{connections: l.connections})

	accept := func() (transport.StreamConn, error) {
		listener, ok := l.wrapped.(*tcpListenerAdapter)
		if !ok {
			return nil, errors.New("wrapped listener is not a tcpListenerAdapter")
		}

		conn, err := listener.AcceptTCP()
		if err == nil {
			conn.SetKeepAlive(true)
		}

		return conn, err
	}

	handler := func(ctx context.Context, conn transport.StreamConn) {
		// Add the client connection to the context so it can be used by the LocalDialer
		ctx = context.WithValue(ctx, clientConnCtxKey{}, conn)
		tcpHandler.Handle(ctx, conn)
	}

	go service.StreamServe(accept, handler)
	return l
}

// clientConnCtxKey is a context key being used to share the client connection
type clientConnCtxKey struct{}

// Accept implements Accept() from net.Listener
func (l *llistener) Accept() (net.Conn, error) {
	select {
	case conn, ok := <-l.connections:
		if !ok {
			return nil, ErrListenerClosed
		}
		return conn, nil
	case <-l.closedSignal:
		return nil, ErrListenerClosed
	}
}

// Close implements Close() from net.Listener
func (l *llistener) Close() error {
	l.closeOnce.Do(func() {
		close(l.closedSignal)
		l.closeError = l.wrapped.Close()
	})
	return l.closeError
}

// Addr implements Addr() from net.Listener
func (l *llistener) Addr() net.Addr {
	return l.wrapped.Addr()
}

// this is an adapter that fulfills the expectation
// of the shadowsocks handler that it can independently
// close the read and write on it's upstream side.
type tcpConnAdapter struct {
	net.Conn
}

func (c *tcpConnAdapter) Wrapped() net.Conn {
	return c.Conn
}

// this is triggered when the remote end is finished.
// This triggers a close of both ends.
func (c *tcpConnAdapter) CloseRead() error {
	tcpConn, ok := c.asTCPConn()
	if ok {
		return tcpConn.CloseRead()
	}
	return c.Close()
}

// this is triggered when a client finishes writing,
// it is handled as a no-op, we just ignore it since
// we don't depend on half closing the connection to
// signal anything.
func (c *tcpConnAdapter) CloseWrite() error {
	tcpConn, ok := c.asTCPConn()
	if ok {
		return tcpConn.CloseWrite()
	}
	return nil
}

func (c *tcpConnAdapter) SetKeepAlive(keepAlive bool) error {
	tcpConn, ok := c.asTCPConn()
	if ok {
		return tcpConn.SetKeepAlive(keepAlive)
	}
	return nil
}

func (c *tcpConnAdapter) asTCPConn() (*net.TCPConn, bool) {
	var tcpConn *net.TCPConn
	netx.WalkWrapped(c, func(conn net.Conn) bool {
		switch t := conn.(type) {
		case *net.TCPConn:
			tcpConn = t
			return false
		}

		// Keep looking
		return true
	})
	return tcpConn, tcpConn != nil
}

type tcpListenerAdapter struct {
	net.Listener
}

func (l *tcpListenerAdapter) AcceptTCP() (TCPConn, error) {
	conn, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}
	return &tcpConnAdapter{conn}, nil
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
	return l.clientTCPConn.(*tcpConnAdapter).Wrapped()
}
