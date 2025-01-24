// Package tlslistener provides a wrapper around tls.Listen that allows
// descending into the wrapped net.Conn
package tlslistener

import (
	"crypto/tls"
	"net"
	"sync"

	"github.com/getlantern/errors"
	"github.com/getlantern/golog"
	"github.com/getlantern/tlsdefaults"

	utls "github.com/refraction-networking/utls"

	"github.com/getlantern/http-proxy-lantern/v2/instrument"
	"github.com/getlantern/http-proxy-lantern/v2/logger"
)

var (
	// log = golog.LoggerFor("tlslistener")
	log = logger.InitializedLogger.SetStdLogger(golog.LoggerFor("tlslistener"))
)

// Wrap wraps the specified listener in our default TLS listener.
func Wrap(wrapped net.Listener, keyFile, certFile, sessionTicketKeyFile, firstSessionTicketKey, sessionTicketKeys string,
	requireSessionTickets bool, missingTicketReaction HandshakeReaction, allowTLS13 bool,
	instrument instrument.Instrument) (net.Listener, error) {

	cfg, err := tlsdefaults.BuildListenerConfig(wrapped.Addr().String(), keyFile, certFile)
	if err != nil {
		return nil, err
	}

	utlsConfig := &utls.Config{}

	// Depending on the ClientHello generated, we use session tickets both for normal
	// session ticket resumption as well as pre-negotiated session tickets as obfuscation.
	// Ideally we'll make this work with TLS 1.3, see:
	// https://github.com/getlantern/lantern-internal/issues/3057
	// https://github.com/getlantern/lantern-internal/issues/3850
	// https://github.com/getlantern/lantern-internal/issues/4111
	if !allowTLS13 {
		cfg.MaxVersion = tls.VersionTLS12
	}

	expectTicketsFromFile := sessionTicketKeyFile != ""
	expectTicketsInMemory := sessionTicketKeys != ""
	expectTickets := expectTicketsFromFile || expectTicketsInMemory

	listener := &tlslistener{
		wrapped:               wrapped,
		cfg:                   cfg,
		log:                   log,
		expectTickets:         expectTickets,
		requireTickets:        requireSessionTickets,
		utlsCfg:               utlsConfig,
		missingTicketReaction: missingTicketReaction,
		instrument:            instrument,
	}

	onKeys := func(keys [][32]byte) {
		cfg.SetSessionTicketKeys(keys)
		utlsConfig.SetSessionTicketKeys(keys)
		listener.ticketKeysMutex.Lock()
		defer listener.ticketKeysMutex.Unlock()
		listener.ticketKeys = make([]utls.TicketKey, 0, len(keys))
		for _, k := range keys {
			listener.ticketKeys = append(listener.ticketKeys, utls.TicketKeyFromBytes(k))
		}
		log.Debug("Finished setting listener keys")
	}

	if expectTicketsFromFile {
		log.Debugf("Will rotate session ticket key and store in %v", sessionTicketKeyFile)
		maintainSessionTicketKeyFile(sessionTicketKeyFile, firstSessionTicketKey, onKeys)
	} else if expectTicketsInMemory {
		log.Debug("Will rotate through session tickets in memory")
		if err := maintainSessionTicketKeysInMemory(sessionTicketKeys, onKeys); err != nil {
			return nil, errors.New("unable to maintain session ticket keys in memory: %v", err)
		}
	}

	return listener, nil
}

type tlslistener struct {
	wrapped               net.Listener
	cfg                   *tls.Config
	log                   golog.Logger
	expectTickets         bool
	requireTickets        bool
	utlsCfg               *utls.Config
	missingTicketReaction HandshakeReaction
	instrument            instrument.Instrument
	ticketKeys            utls.TicketKeys
	ticketKeysMutex       sync.RWMutex
}

func (l *tlslistener) Accept() (net.Conn, error) {
	conn, err := l.wrapped.Accept()
	if err != nil {
		return nil, err
	}
	if !l.expectTickets || !l.requireTickets {
		return &tlsconn{Conn: tls.Server(conn, l.cfg), wrapped: conn}, nil
	}

	helloConn, cfg := newClientHelloRecordingConn(conn, l.cfg, l.utlsCfg, l.getTicketKeys(), l.missingTicketReaction, l.instrument)
	return &tlsconn{Conn: tls.Server(helloConn, cfg), wrapped: conn, helloConn: helloConn}, nil
}

func (l *tlslistener) getTicketKeys() utls.TicketKeys {
	l.ticketKeysMutex.RLock()
	defer l.ticketKeysMutex.RUnlock()
	return l.ticketKeys
}

func (l *tlslistener) Addr() net.Addr {
	return l.wrapped.Addr()
}

func (l *tlslistener) Close() error {
	return l.wrapped.Close()
}

type ProbingDetectingConn interface {
	// If there was suspected probing on this connection, return the error message for that suspected probing
	// (e.g. "ClientHello does not support session tickets"), else return empty string.
	ProbingError() string
}

type tlsconn struct {
	net.Conn
	wrapped   net.Conn
	helloConn *clientHelloRecordingConn
}

func (conn *tlsconn) Wrapped() net.Conn {
	return conn.wrapped
}

func (conn *tlsconn) ProbingError() string {
	if conn.helloConn == nil {
		return ""
	}
	conn.helloConn.helloMutex.Lock()
	err := conn.helloConn.probingError
	conn.helloConn.helloMutex.Unlock()
	return err
}
