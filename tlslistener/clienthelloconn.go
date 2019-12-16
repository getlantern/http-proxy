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
	fullHello    []byte
}

func (rrc *clientHelloRecordingConn) Read(b []byte) (int, error) {
	return rrc.activeReader.Read(b)
}

func (rrc *clientHelloRecordingConn) processHello(info *tls.ClientHelloInfo) (*tls.Config, error) {
	// Skip checking error as net.Addr.String() should be in valid form
	sourceIP, _, _ := net.SplitHostPort(rrc.RemoteAddr().String())

	// The hello is read at this point, so switch to no longer write incoming data to a second buffer.
	rrc.helloMutex.Lock()
	rrc.activeReader = rrc.Conn
	rrc.helloMutex.Unlock()

	// TODO: be a little more memory efficient here?
	rrc.fullHello = rrc.dataRead.Bytes()
	hello := rrc.fullHello[5:]

	defer func() {
		rrc.dataRead.Reset()
		bufferPool.Put(rrc.dataRead)
	}()

	// We use uTLS here purely because it exposes more TLS handshake internals, allowing
	// us to decrypt the ClientHello and session tickets, for example. We use those functions
	// separately without switching to uTLS entirely to allow continued upgrading of the TLS stack
	// as new Go versions are released.
	helloMsg, err := utls.UnmarshalClientHello(hello)

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

	if len(helloMsg.SessionTicket) == 0 {
		return rrc.helloError("ClientHello has no session ticket", sourceIP)
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

	return nil, newNonLanternHelloError(errStr, rrc.fullHello)
	/*
		// TODO: Connect to whatever domain the proxy is mimicking, not whatever the client
		// says to connect to.
		rawUpstreamConn, err := net.DialTimeout("tcp", "microsoft.com:443", 20*time.Second)

		if err != nil {
			rrc.log.Errorf("Could not connect upstream %v", err)
			return nil, err
		}

		defer rrc.Conn.Close()
		defer rawUpstreamConn.Close()
		errc := make(chan error, 1)
		rc := refractionCopier{
			in:  rrc.Conn,
			out: rawUpstreamConn,
		}
		go rc.copyToBackend(errc)
		go rc.copyFromBackend(errc)
		<-errc
		return nil, nil
	*/
}

/*
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
*/
