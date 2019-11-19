package tlslistener

import (
	"bytes"
	"crypto/tls"
	"io"
	"net"
	"sync"

	"github.com/getlantern/golog"
	utls "github.com/getlantern/utls"

	"github.com/getlantern/http-proxy-lantern/instrument"
)

var bufferPool = sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

var disallowLookbackForTesting bool

func newClientHelloRecordingConn(rawConn net.Conn, cfg *tls.Config) (net.Conn, *tls.Config) {
	buf := bufferPool.Get().(*bytes.Buffer)
	cfgClone := cfg.Clone()
	rrc := &clientHelloRecordingConn{
		Conn:         rawConn,
		dataRead:     buf,
		log:          golog.LoggerFor("clienthello-conn"),
		cfg:          cfgClone,
		activeReader: io.TeeReader(rawConn, buf),
		helloMutex:   &sync.Mutex{},
	}
	cfgClone.GetConfigForClient = rrc.processHello

	return rrc, cfgClone
}

type clientHelloRecordingConn struct {
	net.Conn
	dataRead     *bytes.Buffer
	log          golog.Logger
	activeReader io.Reader
	helloMutex   *sync.Mutex
	cfg          *tls.Config
}

func (rrc *clientHelloRecordingConn) Read(b []byte) (int, error) {
	return rrc.activeReader.Read(b)
}

func (rrc *clientHelloRecordingConn) processHello(info *tls.ClientHelloInfo) (*tls.Config, error) {
	// The hello is read at this point, so switch to no longer write incoming data to a second buffer.
	rrc.helloMutex.Lock()
	rrc.activeReader = rrc.Conn
	rrc.helloMutex.Unlock()

	hello := rrc.dataRead.Bytes()[5:]

	// Note we purely use utls here to parse the ClientHello.
	helloMsg, err := utls.UnmarshalClientHello(hello)

	rrc.dataRead.Reset()
	bufferPool.Put(rrc.dataRead)

	// Skip checking error as net.Addr.String() should be in valid form
	sourceIP, _, _ := net.SplitHostPort(rrc.RemoteAddr().String())

	if err != nil {
		instrument.SuspectedProbing(sourceIP, "malformed ClientHello")
		rrc.log.Errorf("Could not parse hello? %v", err)
		return nil, err
	}

	// Allow access from localhost to generate session states to distribute to
	// Lantern clients.
	if !disallowLookbackForTesting && net.ParseIP(sourceIP).IsLoopback() {
		return nil, nil
	}

	newConfig := rrc.cfg.Clone()
	if true { // is TLS 1.2 or below
		// Return an non-existent cipher suite to prevent full handshake if the
		// client doesn't have a valid session ticket.
		newConfig.CipherSuites = []uint16{0} // tls.TLS_RSA_WITH_RC4_128_SHA
	} else {
		newConfig.Certificates = []uint16{Certificate{}} // tls.TLS_RSA_WITH_RC4_128_SHA
	}

	if !helloMsg.TicketSupported {
		errStr := "ClientHello does not support session tickets"
		instrument.SuspectedProbing(sourceIP, errStr)
		rrc.log.Error(errStr)
		return newConfig, nil
	}
	if len(helloMsg.SessionTicket) == 0 {
		errStr := "ClientHello has no session ticket"
		instrument.SuspectedProbing(sourceIP, errStr)
		rrc.log.Error(errStr)
		return newConfig, nil
	}

	return newConfig, nil
}
