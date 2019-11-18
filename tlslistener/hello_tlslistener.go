package tlslistener

import (
	"bytes"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"sync"
	"time"

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
	// Skip checking error as net.Addr.String() should be in valid form
	sourceIP, _, _ := net.SplitHostPort(rrc.RemoteAddr().String())

	// The hello is read at this point, so switch to no longer write incoming data to a second buffer.
	rrc.helloMutex.Lock()
	rrc.activeReader = rrc.Conn
	rrc.helloMutex.Unlock()

	// Skip the handshake record type byte, the protocol version, and the handshake message length.
	hello := rrc.dataRead.Bytes()[5:]

	// Note we purely use utls here to parse the ClientHello.
	helloMsg, err := utls.UnmarshalClientHello(hello)

	rrc.dataRead.Reset()
	bufferPool.Put(rrc.dataRead)

	if err != nil {
		return rrc.helloError("malformed ClientHello", sourceIP, true)
	}

	if !disallowLookbackForTesting && net.ParseIP(sourceIP).IsLoopback() {
		return nil, nil
	}

	if !helloMsg.TicketSupported {
		if info.ServerName == "" {
			return rrc.helloError("ClientHello does not support session tickets", sourceIP, true)
		}
		// TODO: Note we only want to honor SNI to the domains we're actually configured for, as allowing SNI
		// to anywhere would be clearly fingerprintable.
		rawConn, err := net.DialTimeout("tcp", info.ServerName+":443", 10*time.Second)
		if err != nil {
			return rrc.helloError("Could not dial upstream", sourceIP, false)
		}
		cfg := &utls.Config{}
		uconn := utls.UClient(rawConn, cfg, utls.HelloChrome_Auto)
		uconn.HandshakeState = utls.ClientHandshakeState{
			Hello: helloMsg,
			C:     uconn.Conn,
		}
		uconn.ClientHelloBuilt = true
		uconn.Handshake()
		//uconn.SetSessionState(chs)
		//conn.Write(hello)
	}
	if len(helloMsg.SessionTicket) == 0 {
		return rrc.helloError("ClientHello has no session ticket", sourceIP, true)
	}

	rrc.log.Debugf("Session ticket is: %#v", helloMsg.SessionTicket)
	return nil, nil
}

func (rrc *clientHelloRecordingConn) helloError(errStr, sourceIP string, suspicious bool) (*tls.Config, error) {
	if suspicious {
		instrument.SuspectedProbing(sourceIP, errStr)
	}
	rrc.log.Error(errStr)
	return nil, errors.New(errStr)
}
