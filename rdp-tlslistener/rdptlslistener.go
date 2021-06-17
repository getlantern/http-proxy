// Package rdplistener provides a wrapper around tls.Listen that allows
// descending into the wrapped net.Conn (with a RDP twist)
package rdplistener

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"log"
	"net"

	"github.com/getlantern/golog"

	utls "github.com/refraction-networking/utls"

	"github.com/getlantern/http-proxy-lantern/v2/instrument"
)

// Wrap wraps the specified listener in our default TLS listener.
func Wrap(wrapped net.Listener, sessionTicketKeyFile string,
	requireSessionTickets bool, missingTicketReaction HandshakeReaction, instrument instrument.Instrument, reflectionTarget string) (net.Listener, error) {
	cfg, err := BuildListenerConfig()
	if err != nil {
		return nil, err
	}

	log := golog.LoggerFor("lantern-proxy-rdplistener")

	utlsConfig := &utls.Config{}
	onKeys := func(keys [][32]byte) {
		utlsConfig.SetSessionTicketKeys(keys)
	}

	// RDP on windows 7,8,10 does not have TLS 1.3
	cfg.MaxVersion = tls.VersionTLS12

	expectTickets := sessionTicketKeyFile != ""
	if expectTickets {
		log.Debugf("Will rotate session ticket key and store in %v", sessionTicketKeyFile)
		maintainSessionTicketKey(cfg, sessionTicketKeyFile, onKeys)
	}

	listener := &rdptlslistener{
		wrapped:               wrapped,
		cfg:                   cfg,
		log:                   log,
		expectTickets:         expectTickets,
		requireTickets:        requireSessionTickets,
		utlsCfg:               &utls.Config{},
		missingTicketReaction: missingTicketReaction,
		instrument:            instrument,
		reflectionTarget:      reflectionTarget,
		handshakedConnections: make(chan net.Conn),
	}
	go listener.acceptWithHandshakes()
	return listener, nil
}

type rdptlslistener struct {
	wrapped               net.Listener
	cfg                   *tls.Config
	log                   golog.Logger
	expectTickets         bool
	requireTickets        bool
	utlsCfg               *utls.Config
	missingTicketReaction HandshakeReaction
	instrument            instrument.Instrument
	reflectionTarget      string // BBBBBB

	handshakedConnections chan net.Conn
}

var (
	rdpStartTLS     = []byte("\x03\x00\x00\x13\x0e\xe0\x00\x00\x00\x00\x00\x01\x00\x08\x00\x0b\x00\x00\x00")
	rdpAltStartTLS  = []byte("\x03\x00\x00\x13\x0e\xe0\x00\x00\x00\x00\x00\x01\x00\x08\x00\x03\x00\x00\x00")
	rdpStartTLSAck  = []byte("\x03\x00\x00\x13\x0e\xd0\x00\x00\x12\x34\x00\x02\x1f\x08\x00\x08\x00\x00\x00")
	badRDPHandshake = fmt.Errorf("Bad RDP START-TLS Handshake")
)

func (l *rdptlslistener) acceptWithHandshakes() {
	for {
		conn, err := l.wrapped.Accept()
		if err != nil {
			continue
		}

		incomingRdpClientStartTLS := make([]byte, 32)
		n, err := conn.Read(incomingRdpClientStartTLS)
		if err != nil {
			conn.Close()
			return
		}

		if bytes.Compare(incomingRdpClientStartTLS[:n], rdpStartTLS) != 0 &&
			bytes.Compare(incomingRdpClientStartTLS[:n], rdpAltStartTLS) != 0 {
			// "REFLECT"! TODO!! AAAA!
			// return nil, badRDPHandshake
			log.Printf("Fuck? %x", incomingRdpClientStartTLS[:n])
			return
		}

		_, err = conn.Write(rdpStartTLSAck)
		if err != nil {
			conn.Close()
			return
		}

		// This connection is now ready for TLS!
		l.handshakedConnections <- conn
	}
}

func (l *rdptlslistener) fullyReflectRDP(conn net.Conn) {

}

func (l *rdptlslistener) Accept() (net.Conn, error) {
	conn := <-l.handshakedConnections
	// Here we are grabbing pre-handshaked connections
	helloConn, cfg := newClientHelloRecordingConn(conn, l.cfg, l.utlsCfg, l.missingTicketReaction, l.instrument)
	return &rdpconn{tls.Server(helloConn, cfg), conn}, nil
}

func (l *rdptlslistener) Addr() net.Addr {
	return l.wrapped.Addr()
}

func (l *rdptlslistener) Close() error {
	return l.wrapped.Close()
}

type rdpconn struct {
	net.Conn
	wrapped net.Conn
}

func (conn *rdpconn) Wrapped() net.Conn {
	return conn.wrapped
}
