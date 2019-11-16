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
)

var bufferPool = sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

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

	hello := rrc.dataRead.Bytes()

	// Note we purely use utls here to parse the ClientHello.
	helloMsg, err := utls.UnmarshalClientHello(hello)

	rrc.dataRead.Reset()
	bufferPool.Put(rrc.dataRead)

	if err != nil {
		rrc.log.Errorf("Could not parse hello? %v", err)
		return nil, err
	}

	if !helloMsg.TicketSupported || len(helloMsg.SessionTicket) == 0 {
		rrc.log.Error("ClientHello does not support session tickets")
		return nil, errors.New("ClientHello does not support session tickets")
	}

	return nil, nil
}
