package net

import (
	"io"
	"net"
)

// TCPConn is an interface for conns that expose specific functionality from net.TCPConn
type TCPConn interface {
	net.Conn
	// Closes the Read end of the connection, allowing for the release of resources.
	// No more reads should happen.
	CloseRead() error
	// Closes the Write end of the connection. An EOF or FIN signal may be
	// sent to the connection target.
	CloseWrite() error
	// Like SetKeepAlive from net.TCPConn
	SetKeepAlive(bool) error
}

type tcpConnAdaptor struct {
	TCPConn
	r io.Reader
	w io.Writer
}

func (dc *tcpConnAdaptor) Read(b []byte) (int, error) {
	return dc.r.Read(b)
}
func (dc *tcpConnAdaptor) WriteTo(w io.Writer) (int64, error) {
	return io.Copy(w, dc.r)
}
func (dc *tcpConnAdaptor) CloseRead() error {
	return dc.TCPConn.CloseRead()
}
func (dc *tcpConnAdaptor) Write(b []byte) (int, error) {
	return dc.w.Write(b)
}
func (dc *tcpConnAdaptor) ReadFrom(r io.Reader) (int64, error) {
	return io.Copy(dc.w, r)
}
func (dc *tcpConnAdaptor) CloseWrite() error {
	return dc.TCPConn.CloseWrite()
}

// WrapDuplexConn wraps an existing DuplexConn with new Reader and Writer, but
// preserving the original CloseRead() and CloseWrite().
func WrapConn(c TCPConn, r io.Reader, w io.Writer) TCPConn {
	conn := c
	// We special-case duplexConnAdaptor to avoid multiple levels of nesting.
	if a, ok := c.(*tcpConnAdaptor); ok {
		conn = a.TCPConn
	}
	return &tcpConnAdaptor{TCPConn: conn, r: r, w: w}
}

func copyOneWay(leftConn, rightConn TCPConn) (int64, error) {
	n, err := io.Copy(leftConn, rightConn)
	// Send FIN to indicate EOF
	leftConn.CloseWrite()
	// Release reader resources
	rightConn.CloseRead()
	return n, err
}

// Relay copies between left and right bidirectionally. Returns number of
// bytes copied from right to left, from left to right, and any error occurred.
// Relay allows for half-closed connections: if one side is done writing, it can
// still read all remaining data from its peer.
func Relay(leftConn, rightConn TCPConn) (int64, int64, error) {
	type res struct {
		N   int64
		Err error
	}
	ch := make(chan res)

	go func() {
		n, err := copyOneWay(rightConn, leftConn)
		ch <- res{n, err}
	}()

	n, err := copyOneWay(leftConn, rightConn)
	rs := <-ch

	if err == nil {
		err = rs.Err
	}
	return n, rs.N, err
}

type ConnectionError struct {
	// TODO: create status enums and move to metrics.go
	Status  string
	Message string
	Cause   error
}

func NewConnectionError(status, message string, cause error) *ConnectionError {
	return &ConnectionError{Status: status, Message: message, Cause: cause}
}

// Interface for something like net.TCPListener but not using the concrete type.
type TCPListener interface {
	net.Listener

	AcceptTCP() (TCPConn, error)

	Close() error

	Addr() net.Addr
}

type tcpDuplexListenerAdapter struct {
	*net.TCPListener
}

func (l *tcpDuplexListenerAdapter) AcceptTCP() (TCPConn, error) {
	return l.TCPListener.AcceptTCP()
}

func AdaptListener(l *net.TCPListener) TCPListener {
	return &tcpDuplexListenerAdapter{l}
}
