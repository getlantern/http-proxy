// Package tlslistener provides a wrapper around tls.Listen that allows
// descending into the wrapped net.Conn
package tlslistener

import (
	"bytes"
	"crypto/tls"
	"io"
	"net"
	"sync"
	"time"

	"github.com/getlantern/golog"
	"github.com/getlantern/tlsdefaults"

	utls "github.com/getlantern/utls"
)

func newNonLanternHelloError(msg string, hello []byte) *nonLanternHelloError {
	return &nonLanternHelloError{
		msg:   msg,
		hello: hello,
	}
}

type nonLanternHelloError struct {
	msg   string
	hello []byte
}

func (nonLanternHelloError) Error() string { return "tls: no Lantern session ticket in ClientHello" }

// Wrap wraps the specified listener in our default TLS listener.
func Wrap(wrapped net.Listener, keyFile string, certFile string, sessionTicketKeyFile string,
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
		sessionticket.maintainSessionTicketKey(cfg, sessionTicketKeyFile, onKeys)
	}

	listener := &tlslistener{wrapped, cfg, log, expectTickets, requireSessionTickets, utlsConfig}
	return listener, nil
}

type tlslistener struct {
	wrapped        net.Listener
	cfg            *tls.Config
	log            golog.Logger
	expectTickets  bool
	requireTickets bool
	utlsCfg        *utls.Config
}

func (l *tlslistener) Accept() (net.Conn, error) {
	conn, err := l.wrapped.Accept()
	if err != nil {
		return nil, err
	}
	/*
		if !l.expectTickets || !l.requireTickets {
			return l.newRefractionConn(tls.Server(conn, l.cfg), conn), nil
		}
	*/
	helloConn, cfg := newClientHelloRecordingConn(conn, l.cfg, l.utlsCfg)
	return l.newRefractionConn(tls.Server(helloConn, cfg), conn), nil
}

func (l *tlslistener) Addr() net.Addr {
	return l.wrapped.Addr()
}

func (l *tlslistener) Close() error {
	return l.wrapped.Close()
}

func (l *tlslistener) newRefractionConn(tlsConn *tls.Conn, rawConn net.Conn) *tlsconn {
	return &tlsconn{
		tlsConn,
		bufferPool.Get().(*bytes.Buffer),
		rawConn,
		tlsConn,
		&sync.Mutex{},
		l.log,
	}
}

type tlsconn struct {
	*tls.Conn
	dataRead  *bytes.Buffer
	wrapped   net.Conn
	active    net.Conn
	connMutex *sync.Mutex
	log       golog.Logger
}

func (conn *tlsconn) Read(b []byte) (int, error) {
	conn.log.Debug("Reading")
	if err := conn.Handshake(); err != nil {
		return 0, err
	}
	if len(b) == 0 {
		// Put this after Handshake, in case people were calling
		// Read(nil) for the side effect of the Handshake.
		return 0, nil
	}
	return conn.active.Read(b)
}

func (conn *tlsconn) Write(b []byte) (int, error) {
	conn.log.Debug("Writing")
	return conn.active.Write(b)
}

//func (conn *tlsconn) readClientHello() error {
//}

// Strategy:
// 1) Read Client Hello
// 2) If it's ours, keep operating over TLS re-feed the hello into a new TLS connection
// 3) If it's not ours, just re-send the client hello upstream and relay all traffic.

func (conn *tlsconn) Handshake() error {
	conn.log.Debug("Handhsakings")
	if err := conn.Conn.Handshake(); err != nil {
		conn.log.Debugf("Handshake error: %v", err)
		if nonLantern, ok := err.(*nonLanternHelloError); ok {
			conn.log.Debugf("Got non lantern hello error!! %v", nonLantern)
			//if err == errNonLanternHello {
			// If we get a nonLanternHelloError, that means something other than Lantern,
			// potentially a censor's probe, is trying to connect. Under this scheme we want to
			// begin relaying traffic to the destination site in that case. To do this, we must:
			//
			// 1. Establish a connection to the destination site
			// 2. Write the ClientHello to the destination
			// 3. Relay all future traffic in either direction.

			// TODO: Get the configured domain
			rawUpstreamConn, err := net.DialTimeout("tcp", "microsoft.com:443", 20*time.Second)
			if err != nil {
				conn.log.Errorf("Could not connect upstream %v", err)
				return err
			}

			conn.connMutex.Lock()
			conn.active = &readCopyConn{conn.wrapped, rawUpstreamConn}
			conn.connMutex.Unlock()

			errc := make(chan error, 1)
			rc := refractionCopier{
				in:  conn.Conn,
				out: rawUpstreamConn,
			}
			//go rc.copyToBackend(errc)
			go func() {
				defer conn.Conn.Close()
				defer rawUpstreamConn.Close()
				rc.copyFromBackend(errc)
				<-errc
			}()
			rawUpstreamConn.Write(nonLantern.hello)
			return nil
		}
		return err
	}
	return nil
}

func (conn *tlsconn) Wrapped() net.Conn {
	return conn.wrapped
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

type readCopyConn struct {
	net.Conn
	out net.Conn
}

func (rc *readCopyConn) Read(b []byte) (int, error) {
	n, err := rc.Conn.Read(b)
	if err != nil {
		return n, err
	}
	return rc.out.Write(b)
}

func (rc *readCopyConn) Write(b []byte) (int, error) {
	return rc.Conn.Write(b)
}
