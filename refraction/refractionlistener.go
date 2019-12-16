// Package refraction performs real TLS handshakes and proxies real traffic for spoofed domains
// while detecting Lantern clients and proxying traffic for them normally following the
// handshake.
package refraction

import (
	"crypto/tls"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/getlantern/golog"
	"github.com/getlantern/tlsdefaults"

	utls "github.com/getlantern/utls"
)

// Wrap wraps the specified listener in our default refraction listener. The SNI field
// is the expected domain in incoming ClientHellos and is the upstream domain we should
// connect to.
func Wrap(wrapped net.Listener, sni string, keyFile string, certFile string, sessionTicketKeyFile string,
	requireSessionTickets bool) (net.Listener, error) {
	cfg, err := tlsdefaults.BuildListenerConfig(wrapped.Addr().String(), keyFile, certFile)
	if err != nil {
		return nil, err
	}
	// This is a bit of a hack to make pre-shared TLS sessions work with uTLS. Ideally we'll make this
	// work with TLS 1.3, see https://github.com/getlantern/lantern-internal/issues/3057.
	cfg.MaxVersion = tls.VersionTLS12

	log := golog.LoggerFor("lantern-proxy-tlslistener")

	utlsConfig := &utls.Config{}
	onKeys := func(keys [][32]byte) {
		utlsConfig.SetSessionTicketKeys(keys)
	}
	expectTickets := sessionTicketKeyFile != ""
	if expectTickets {
		log.Debugf("Will rotate session ticket key and store in %v", sessionTicketKeyFile)
		tlslistener.maintainSessionTicketKey(cfg, sessionTicketKeyFile, onKeys)
	}
	return newRefractionListener(wrapped, sni), nil
}

func newRefractionListener(wrapped net.Listener, sni string) net.Listener {
	return &refractionlistener{wrapped, golog.LoggerFor("refraction-listener"), sni}
}

type refractionlistener struct {
	wrapped net.Listener
	log     golog.Logger
	sni     string
}

func (l *refractionlistener) Accept() (net.Conn, error) {
	conn, err := l.wrapped.Accept()
	if err != nil {
		return nil, err
	}
	rc := l.newRefractionConn(conn)
	return rc, nil
}

func (l *refractionlistener) Addr() net.Addr {
	return l.wrapped.Addr()
}

func (l *refractionlistener) Close() error {
	return l.wrapped.Close()
}

func (l *refractionlistener) newRefractionConn(conn net.Conn) net.Conn {
	rc := &refractionconn{
		conn: conn,
		log:  l.log,
		rl:   l,
	}
	return rc
}

type refractionconn struct {
	conn net.Conn
	log  golog.Logger
	rl   *refractionlistener
	// helloRead is 1 if we've read the client hello.
	helloRead  uint32
	helloMutex sync.Mutex
}

func (c *refractionconn) Read(b []byte) (int, error) {
	c.readHello()
	if len(b) == 0 {
		// Put this after reading hello  in case people were calling Read(nil) for the side effect
		// of reading the hello, such as in a test.
		return 0, nil
	}

	return c.conn.Read(b)

}

func (c *refractionconn) readHello() {
	c.helloMutex.Lock()
	defer c.helloMutex.Unlock()
	if c.isHelloRead() {
		return
	}

	tlsConn := utls.NewClientConn(c.conn)
	helloMsg, err := tlsConn.ReadClientHelloRaw()
	if err != nil {
		c.log.Errorf("Could not parse hello %v", err)
		c.conn.Close()
	}

	atomic.StoreUint32(&c.helloRead, 1)

	// If the client is Lantern, we need to switch to TLS resumption.
	if c.isLantern(helloMsg) {

	} else {
		// If the client is not Lantern, just relay to the destination site. Note that we don't want
		// to do things like double check for SNI mismatches, configure idle timings, etc, as we
		// want to do exactly what the destination site does in those cases.
		c.log.Debug("Relaying")
		go c.relay(helloMsg)
	}
}

func (c *refractionconn) isHelloRead() bool {
	return atomic.LoadUint32(&c.helloRead) == 1
}

func (c *refractionconn) isLantern(helloMsg *utls.ClientHelloMsg) bool {
	// We want to make sure that the client is using resumption with one of our
	// pre-defined tickets. If it doesn't we should again return some sort of error
	// or just close the connection.
	if !helloMsg.TicketSupported {
		c.log.Debug("Tickets not supported")
		return false
	}

	if len(helloMsg.SessionTicket) == 0 {
		c.log.Debug("No session tickets")
		return false
	}

	/*
		plainText, _ := utls.DecryptTicketWith(helloMsg.SessionTicket, rrc.utlsCfg)
		if plainText == nil || len(plainText) == 0 {
		}
	*/
	return true
}

func (c *refractionconn) relay(hello *utls.ClientHelloMsg) {
	defer c.conn.Close()

	rawUpstreamConn, err := net.DialTimeout("tcp", c.rl.sni, 20*time.Second)
	if err != nil {
		c.log.Errorf("Could not connect upstream %v", err)
		return
	}
	defer rawUpstreamConn.Close()
	tlsUpstreamConn := utls.NewClientConn(rawUpstreamConn)
	if _, err := tlsUpstreamConn.WriteRecord(22, hello.Marshall()); err != nil {
		return
	}

	errc := make(chan error, 1)
	rc := refractionCopier{
		in:  c,
		out: rawUpstreamConn,
	}

	go rc.copyFromBackend(errc)
	go rc.copyToBackend(errc)
	<-errc
}

// refractionCopier exists so goroutines proxying data have nice names in stacks.
type refractionCopier struct {
	in, out io.ReadWriter
}

func (c refractionCopier) copyFromBackend(errc chan<- error) {
	_, err := io.Copy(c.in, c.out)
	errc <- err
}

func (c refractionCopier) copyToBackend(errc chan<- error) {
	_, err := io.Copy(c.out, c.in)
	errc <- err
}

// LocalAddr returns the local network address.
func (c *refractionconn) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

// RemoteAddr returns the remote network address.
func (c *refractionconn) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

// SetDeadline sets the read and write deadlines associated with the connection.
// A zero value for t means Read and Write will not time out.
// After a Write has timed out, the TLS state is corrupt and all future writes will return the same error.
func (c *refractionconn) SetDeadline(t time.Time) error {
	return c.conn.SetDeadline(t)
}

// SetReadDeadline sets the read deadline on the underlying connection.
// A zero value for t means Read will not time out.
func (c *refractionconn) SetReadDeadline(t time.Time) error {
	return c.conn.SetReadDeadline(t)
}

// SetWriteDeadline sets the write deadline on the underlying connection.
// A zero value for t means Write will not time out.
// After a Write has timed out, the TLS state is corrupt and all future writes will return the same error.
func (c *refractionconn) SetWriteDeadline(t time.Time) error {
	return c.conn.SetWriteDeadline(t)
}

// Close closes the connection.
func (c *refractionconn) Close() error {
	return c.conn.Close()
}

// Write writes data to the connection.
func (c *refractionconn) Write(b []byte) (int, error) {
	return c.conn.Write(b)
}
