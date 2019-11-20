package tlslistener

import (
	"bytes"
	"crypto/tls"
	"errors"
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

func newClientHelloRecordingConn(rawConn net.Conn, cfg *tls.Config, utlsCfg *utls.Config) (net.Conn, *tls.Config) {
	buf := bufferPool.Get().(*bytes.Buffer)
	cfgClone := cfg.Clone()
	rrc := &clientHelloRecordingConn{
		Conn:         rawConn,
		dataRead:     buf,
		log:          golog.LoggerFor("clienthello-conn"),
		cfg:          cfgClone,
		activeReader: io.TeeReader(rawConn, buf),
		helloMutex:   &sync.Mutex{},
		utlsCfg:      utlsCfg,
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
	utlsCfg      *utls.Config
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

	// We use uTLS here purely because it exposes more TLS handshake internals, allowing
	// us to decrypt the ClientHello and session tickets, for example. We use those functions
	// separately without switching to uTLS entirely to allow continued upgrading of the TLS stack
	// as new Go versions are released.
	helloMsg, err := utls.UnmarshalClientHello(hello)

	rrc.dataRead.Reset()
	bufferPool.Put(rrc.dataRead)

	// Skip checking error as net.Addr.String() should be in valid form
	sourceIP, _, _ := net.SplitHostPort(rrc.RemoteAddr().String())

	if err != nil {
		return rrc.helloError("malformed ClientHello", sourceIP)
	}

	// We allow loopback to generate session states (makesessions) to
	// distribute to Lantern clients.
	if !disallowLookbackForTesting && net.ParseIP(sourceIP).IsLoopback() {
		return nil, nil
	}

	// Otherwise, we want to make sure that the client is using resumption with one of our
	// pre-defined tickets. If it doesn't we should again return some sort of error or just
	// close the connection.
	if !helloMsg.TicketSupported {
		return rrc.helloError("ClientHello does not support session tickets", sourceIP)
	}

	plainText, _ := utls.DecryptTicketWith(helloMsg.SessionTicket, rrc.utlsCfg)
	if plainText == nil || len(plainText) == 0 {
		return rrc.helloError("ClientHello has invalid session ticket", sourceIP)
	}

	return nil, nil
}

func (rrc *clientHelloRecordingConn) helloError(errStr, sourceIP string) (*tls.Config, error) {
	instrument.SuspectedProbing(sourceIP, errStr)
	rrc.log.Error(errStr)
	return nil, errors.New(errStr)
}
