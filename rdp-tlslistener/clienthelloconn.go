package rdplistener

import (
	"bytes"
	"crypto/tls"
	"io"
	"log"
	"net"
	"sync"

	"github.com/getlantern/golog"
	"github.com/getlantern/netx"
	"github.com/getlantern/preconn"
	utls "github.com/refraction-networking/utls"

	"github.com/getlantern/http-proxy-lantern/v2/instrument"
)

var (
	reflectBufferSize = 2 << 11 // 4K
)

// HandshakeReaction represents various reactions after seeing certain type of
// TLS ClientHellos, usually indicating active probing.
type HandshakeReaction struct {
	action     string
	getConfig  func(*tls.Config) (*tls.Config, error)
	handleConn func(c *clientHelloRecordingConn)
}

func (hr HandshakeReaction) Action() string {
	return hr.action
}

var (

	// ReflectToRDP dials TLS connection to the designated site and copies
	// everything including the ClientHello back and forth between the client
	// and the site, pretending to be the site itself. It closes the client
	// connection if unable to dial the site.
	ReflectToRDP = func(site string) HandshakeReaction {
		return HandshakeReaction{
			action: "ReflectToRDP",
			handleConn: func(c *clientHelloRecordingConn) {
				defer c.Close()
				upstream, err := net.Dial("tcp", net.JoinHostPort(site, "3389"))
				if err != nil {
					return
				}
				defer upstream.Close()

				upstream.Write(rdpStartTLS) // client RDP-TLS open (like START-TLS)

				// Check if the target upstream server is going to actually do RDP TLS
				upstreamRDPhandshake := make([]byte, 32)
				n, err := upstream.Read(upstreamRDPhandshake)
				if err != nil {
					return
				}

				if bytes.Compare(upstreamRDPhandshake[:n], rdpStartTLSAck) != 0 {
					// Reflection Target Server sent something very strange, be very afraid, just abort.
					return
				}

				// We need to inject the TLS Client Hello back in!

				pConn := preconn.Wrap(c, c.helloPacket)
				suspectedProbeConn := tls.Server(pConn, &tls.Config{Certificates: c.cfg.Certificates})
				err = suspectedProbeConn.Handshake()
				if err != nil {
					log.Printf("Fuck? %v", err)
					return
				}

				upstreamRDPTLS := tls.Client(upstream, &tls.Config{InsecureSkipVerify: true})
				upstreamRDPTLS.Handshake()
				if err != nil {
					return
				}

				bufOut := bytePool.Get().([]byte)
				defer bytePool.Put(bufOut)
				bufIn := bytePool.Get().([]byte)
				defer bytePool.Put(bufIn)
				_, _ = netx.BidiCopy(suspectedProbeConn, upstreamRDPTLS, bufOut, bufIn)
			}}
	}

	// None doesn't react.
	None = HandshakeReaction{
		action: "",
		getConfig: func(c *tls.Config) (*tls.Config, error) {
			return c, nil
		}}
)

var disallowLookbackForTesting bool

var bufferPool = sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	}}

var bytePool = sync.Pool{
	New: func() interface{} {
		return make([]byte, reflectBufferSize)
	}}

func newClientHelloRecordingConn(rawConn net.Conn, cfg *tls.Config, utlsCfg *utls.Config, missingTicketReaction HandshakeReaction, instrument instrument.Instrument) (net.Conn, *tls.Config) {
	buf := bufferPool.Get().(*bytes.Buffer)
	cfgClone := cfg.Clone()
	rrc := &clientHelloRecordingConn{
		Conn:                  rawConn,
		dataRead:              buf,
		log:                   golog.LoggerFor("rdp-conn"),
		cfg:                   cfgClone,
		activeReader:          io.TeeReader(rawConn, buf),
		helloMutex:            &sync.Mutex{},
		utlsCfg:               utlsCfg,
		missingTicketReaction: missingTicketReaction,
		instrument:            instrument,
	}
	cfgClone.GetConfigForClient = rrc.processHello

	return rrc, cfgClone
}

type clientHelloRecordingConn struct {
	net.Conn
	dataRead              *bytes.Buffer
	log                   golog.Logger
	activeReader          io.Reader
	helloMutex            *sync.Mutex
	cfg                   *tls.Config
	utlsCfg               *utls.Config
	missingTicketReaction HandshakeReaction
	instrument            instrument.Instrument
	helloPacket           []byte
}

func (rrc *clientHelloRecordingConn) Read(b []byte) (int, error) {
	return rrc.activeReader.Read(b)
}

func (rrc *clientHelloRecordingConn) processHello(info *tls.ClientHelloInfo) (*tls.Config, error) {
	// The hello is read at this point, so switch to no longer write incoming data to a second buffer.
	rrc.helloMutex.Lock()
	rrc.activeReader = rrc.Conn
	rrc.helloMutex.Unlock()
	defer func() {
		rrc.dataRead.Reset()
		bufferPool.Put(rrc.dataRead)
	}()

	fullHello := rrc.dataRead.Bytes()
	hello := fullHello[5:]
	// We use uTLS here purely because it exposes more TLS handshake internals, allowing
	// us to decrypt the ClientHello and session tickets, for example. We use those functions
	// separately without switching to uTLS entirely to allow continued upgrading of the TLS stack
	// as new Go versions are released.
	helloMsg, err := utls.UnmarshalClientHello(hello)

	rrc.helloPacket = make([]byte, len(fullHello))
	copy(rrc.helloPacket, fullHello)
	if err != nil {
		return rrc.helloError("malformed ClientHello")
	}

	sourceIP := rrc.RemoteAddr().(*net.TCPAddr).IP
	// We allow loopback to generate session states (makesessions) to
	// distribute to Lantern clients.
	if !disallowLookbackForTesting && sourceIP.IsLoopback() {
		return nil, nil
	}

	// Otherwise, we want to make sure that the client is using resumption with one of our
	// pre-defined tickets. If it doesn't we should again return some sort of error or just
	// close the connection.
	if !helloMsg.TicketSupported {
		return rrc.helloError("ClientHello does not support session tickets")
	}

	if len(helloMsg.SessionTicket) == 0 {
		return rrc.helloError("ClientHello has no session ticket")
	}

	plainText, _ := utls.DecryptTicketWith(helloMsg.SessionTicket, rrc.utlsCfg)
	if plainText == nil || len(plainText) == 0 {
		return rrc.helloError("ClientHello has invalid session ticket")
	}

	return nil, nil
}

func (rrc *clientHelloRecordingConn) helloError(errStr string) (*tls.Config, error) {
	sourceIP := rrc.RemoteAddr().(*net.TCPAddr).IP
	rrc.instrument.SuspectedProbing(sourceIP, errStr)
	if rrc.missingTicketReaction.handleConn != nil {
		rrc.missingTicketReaction.handleConn(rrc)
		// at this point the connection has already been closed, returning
		// whatever to the caller is okay.
		return nil, nil
	}
	return rrc.missingTicketReaction.getConfig(rrc.cfg)
}
