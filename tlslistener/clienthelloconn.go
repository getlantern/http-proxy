package tlslistener

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	utls "github.com/refraction-networking/utls"

	"github.com/getlantern/golog"
	"github.com/getlantern/netx"

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
	// AlertHandshakeFailure responds TLS alert 40 (Handshake failure).
	AlertHandshakeFailure = HandshakeReaction{
		action: "AlertHandshakeFailure",
		getConfig: func(c *tls.Config) (*tls.Config, error) {
			clone := c.Clone()
			clone.CipherSuites = []uint16{}
			return clone, nil
		}}

	// AlertProtocolVersion responds TLS alert 70 (Protocol version).
	AlertProtocolVersion = HandshakeReaction{
		action: "AlertProtocolVersion",
		getConfig: func(c *tls.Config) (*tls.Config, error) {
			clone := c.Clone()
			clone.MaxVersion = 1
			return clone, nil
		}}

	// AlertInternalError responds TLS alert 80 (Internal error).
	AlertInternalError = HandshakeReaction{
		action: "AlertInternalError",
		getConfig: func(c *tls.Config) (*tls.Config, error) {
			return nil, errors.New("whatever")
		}}

	// CloseConnection closes the TLS connection arbitrarily.
	CloseConnection = HandshakeReaction{
		action: "CloseConnection",
		handleConn: func(c *clientHelloRecordingConn) {
			c.Close()
		}}

	// ReflectToSite dials TLS connection to the designated site and copies
	// everything including the ClientHello back and forth between the client
	// and the site, pretending to be the site itself. It closes the client
	// connection if unable to dial the site.
	ReflectToSite = func(site string) HandshakeReaction {
		return HandshakeReaction{
			action: "ReflectToSite",
			handleConn: func(c *clientHelloRecordingConn) {
				defer c.Close()
				upstream, err := net.Dial("tcp", net.JoinHostPort(site, "443"))
				if err != nil {
					return
				}
				defer upstream.Close()
				_, err = c.dataRead.WriteTo(upstream)
				if err != nil {
					return
				}
				bufOut := bytePool.Get().([]byte)
				defer bytePool.Put(bufOut)
				bufIn := bytePool.Get().([]byte)
				defer bytePool.Put(bufIn)
				_, _ = netx.BidiCopy(c, upstream, bufOut, bufIn)
			}}
	}

	// None doesn't react.
	None = HandshakeReaction{
		action: "",
		getConfig: func(c *tls.Config) (*tls.Config, error) {
			return c, nil
		}}
)

// Delayed takes a HandshakeReaction and delays d before executing the action.
func Delayed(d time.Duration, r HandshakeReaction) HandshakeReaction {
	r2 := HandshakeReaction{
		action: fmt.Sprintf("%s(after %v)", r.action, d),
	}

	if r.getConfig != nil {
		r2.getConfig = func(c *tls.Config) (*tls.Config, error) {
			time.Sleep(d)
			return r.getConfig(c)
		}
	}
	if r.handleConn != nil {
		r2.handleConn = func(c *clientHelloRecordingConn) {
			time.Sleep(d)
			r.handleConn(c)
		}
	}
	return r2
}

var allowLoopbackForTesting bool

var bufferPool = sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	}}

var bytePool = sync.Pool{
	New: func() interface{} {
		return make([]byte, reflectBufferSize)
	}}

func newClientHelloRecordingConn(rawConn net.Conn, cfg *tls.Config, utlsCfg *utls.Config, ticketKeys utls.TicketKeys, missingTicketReaction HandshakeReaction, instrument instrument.Instrument) (*clientHelloRecordingConn, *tls.Config) {
	buf := bufferPool.Get().(*bytes.Buffer)
	cfgClone := cfg.Clone()
	rrc := &clientHelloRecordingConn{
		Conn:                  rawConn,
		dataRead:              buf,
		log:                   golog.LoggerFor("clienthello-conn"),
		cfg:                   cfgClone,
		ticketKeys:            ticketKeys,
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
	ticketKeys            utls.TicketKeys
	missingTicketReaction HandshakeReaction
	instrument            instrument.Instrument
	probingError          string
}

func (rrc *clientHelloRecordingConn) Read(b []byte) (int, error) {
	return rrc.activeReader.Read(b)
}

// This function concatenates TLS record fragments into a single record.
// The output is the concatenation of the payloads of the input records plus a header.
// The input data is expected to be one or multiple TLS records, each with the following format:
//   - 1 byte: content type
//   - 2 bytes: protocol version
//   - 2 bytes: length of the payload
//   - N bytes: payload
func concatenateTlsRecordsFragments(data []byte) ([]byte, error) {
	const headerLength = 5

	if len(data) < headerLength {
		return nil, fmt.Errorf("input data is too short to contain even one TLS record header")
	}

	totalPayload := []byte{}
	var contentType byte
	var protocolVersion [2]byte

	for i := 0; i < len(data); {
		if len(data[i:]) < headerLength {
			return nil, fmt.Errorf("incomplete TLS record header at position %d", i)
		}

		header := data[i : i+headerLength]
		contentType = header[0]
		protocolVersion = [2]byte{header[1], header[2]}
		length := int(header[3])<<8 | int(header[4])

		if len(data[i+headerLength:]) < length {
			return nil, fmt.Errorf("incomplete TLS record payload at position %d", i+headerLength)
		}

		payload := data[i+headerLength : i+headerLength+length]
		totalPayload = append(totalPayload, payload...)

		i += headerLength + length
	}

	// Construct the new single TLS record
	totalLength := len(totalPayload)
	if totalLength > 0xFFFF {
		return nil, fmt.Errorf("concatenated payload length %d exceeds maximum TLS record size", totalLength)
	}

	header := []byte{contentType, protocolVersion[0], protocolVersion[1], byte(totalLength >> 8), byte(totalLength & 0xFF)}
	result := append(header, totalPayload...)
	return result, nil
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

	concatenatedRecords, err := concatenateTlsRecordsFragments(rrc.dataRead.Bytes())
	if err != nil {
		return rrc.helloError("malformed ClientHello: " + err.Error())
	}

	hello := concatenatedRecords[5:]
	// We use uTLS here purely because it exposes more TLS handshake internals, allowing
	// us to decrypt the ClientHello and session tickets, for example. We use those functions
	// separately without switching to uTLS entirely to allow continued upgrading of the TLS stack
	// as new Go versions are released.
	helloMsg := utls.UnmarshalClientHello(hello)

	if helloMsg == nil {
		return rrc.helloError("malformed ClientHello")
	}

	sourceIP := rrc.RemoteAddr().(*net.TCPAddr).IP
	// We allow loopback to generate session states (makesessions) to
	// distribute to Lantern clients.
	if !allowLoopbackForTesting && sourceIP.IsLoopback() {
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

	uss, err := rrc.utlsCfg.DecryptTicket(helloMsg.SessionTicket, utls.ConnectionState{})
	if err != nil || uss == nil {
		return rrc.helloError("ClientHello has invalid session ticket")
	}

	return nil, nil
}

func (rrc *clientHelloRecordingConn) helloError(errStr string) (*tls.Config, error) {
	sourceIP := rrc.RemoteAddr().(*net.TCPAddr).IP
	rrc.probingError = errStr
	rrc.instrument.SuspectedProbing(context.Background(), sourceIP, errStr)
	if rrc.missingTicketReaction.handleConn != nil {
		rrc.missingTicketReaction.handleConn(rrc)
		// at this point the connection has already been closed, returning
		// whatever to the caller is okay.
		return nil, nil
	}
	return rrc.missingTicketReaction.getConfig(rrc.cfg)
}
