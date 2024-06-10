package shadowsocks

import (
	"context"
	"errors"
	"net"
	"time"

	"github.com/getlantern/golog"
	"github.com/getlantern/netx"

	"github.com/Jigsaw-Code/outline-sdk/transport"
	onet "github.com/Jigsaw-Code/outline-ss-server/net"
	"github.com/Jigsaw-Code/outline-ss-server/service"
)

// shadowsocks/local.go houses adapters for use with Lantern. This mostly is in
// place to allow the Lantern backend to handle upstream connections itself and
// have shadowsocks behave like other transports we use in Lantern.

var (
	log = golog.LoggerFor("shadowsocks")

	ErrListenerClosed = errors.New("listener closed")
)

// This value is lifted from the the main server.go to match behavior
// 59 seconds is most common timeout for servers that do not respond to invalid requests
const tcpReadTimeout time.Duration = 59 * time.Second

// HandleLocalPredicate is a type of function that determines whether to handle an
// upstream address locally or not.  If the function returns true, the address is
// handled locally.  If the funtion returns false, the address is handled by the
// default upstream dial.
type HandleLocalPredicate func(addr string) bool

// AlwaysLocal is a HandleLocalPredicate that requests local handling for all addresses
func AlwaysLocal(addr string) bool { return true }

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
	tcpHandler.SetTargetDialer(&LocalDialer{Listener: l})
	handler := &Handler{TCPHandler: tcpHandler}

	accept := func() (transport.StreamConn, error) {
		switch l.wrapped.(type) {
		case *tcpListenerAdapter, *net.TCPListener:
			conn, err := l.wrapped.(*tcpListenerAdapter).AcceptTCP()
			if err == nil {
				conn.SetKeepAlive(true)
			}
			return conn, err
		default:
			return nil, errors.New("unsupported listener type")
		}
	}

	go service.StreamServe(accept, handler.Handle)
	return l
}

// ClientConnCtxKey is a context key being used to share the client connection
type ClientConnCtxKey struct{}

// Handler implements the service.TCPHandler with some customized operations
// before calling the actual TCPHandler
type Handler struct {
	TCPHandler service.TCPHandler
}

func (h *Handler) Handle(ctx context.Context, conn transport.StreamConn) {
	// Add the client connection to the context so it can be used by the LocalDialer
	ctx = context.WithValue(ctx, ClientConnCtxKey{}, conn)
	h.TCPHandler.Handle(ctx, conn)
}

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
	conn net.Conn
}

func (c *tcpConnAdapter) Wrapped() net.Conn {
	return c.conn
}

func (c *tcpConnAdapter) Close() error {
	return c.conn.Close()
}

func (c *tcpConnAdapter) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

func (c *tcpConnAdapter) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

func (c *tcpConnAdapter) Read(b []byte) (n int, err error) {
	return c.conn.Read(b)
}

func (c *tcpConnAdapter) SetDeadline(t time.Time) error {
	return c.conn.SetDeadline(t)
}

// Write writes data to the connection.
// Write can be made to time out and return an error after a fixed
// time limit; see SetDeadline and SetWriteDeadline.
func (c *tcpConnAdapter) Write(b []byte) (n int, err error) {
	return c.conn.Write(b)
}

// SetReadDeadline sets the deadline for future Read calls
// and any currently-blocked Read call.
// A zero value for t means Read will not time out.
func (c *tcpConnAdapter) SetReadDeadline(t time.Time) error {
	return c.conn.SetReadDeadline(t)
}

// SetWriteDeadline sets the deadline for future Write calls
// and any currently-blocked Write call.
// Even if write times out, it may return n > 0, indicating that
// some of the data was successfully written.
// A zero value for t means Write will not time out.
func (c *tcpConnAdapter) SetWriteDeadline(t time.Time) error {
	return c.conn.SetWriteDeadline(t)
}

// this is triggered when the remote end is finished.
// This triggers a close of both ends.
func (c *tcpConnAdapter) CloseRead() error {
	tcpConn, ok := c.asTCPConn()
	if ok {
		return tcpConn.CloseRead()
	}
	return c.conn.Close()
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
	netx.WalkWrapped(c.conn, func(conn net.Conn) bool {
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
