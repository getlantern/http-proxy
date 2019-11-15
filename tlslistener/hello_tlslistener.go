package tlslistener

import (
	"bytes"
	"crypto/tls"
	"io"
	"net"
	"sync"

	"github.com/getlantern/golog"
	ltls "github.com/getlantern/ltls"
)

func newClientHelloRecordingConn(rawConn net.Conn, cfg *tls.Config, expectTickets bool) (net.Conn, *tls.Config) {
	// TODO: Possibly use sync.Pool here?
	var buf bytes.Buffer
	cfgClone := cfg.Clone()
	rrc := &clientHelloRecordingConn{
		Conn:          rawConn,
		dataRead:      &buf,
		log:           golog.LoggerFor("clienthello-conn"),
		cfg:           cfgClone,
		activeReader:  io.TeeReader(rawConn, &buf),
		expectTickets: expectTickets,
		helloMutex:    &sync.Mutex{},
	}
	cfgClone.GetConfigForClient = rrc.processHello

	return rrc, cfgClone
}

type clientHelloRecordingConn struct {
	net.Conn
	dataRead      *bytes.Buffer
	log           golog.Logger
	activeReader  io.Reader
	helloMutex    *sync.Mutex
	cfg           *tls.Config
	expectTickets bool
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
	helloMsg, err := ltls.UnmarshalClientHello(hello)

	if err != nil || !helloMsg.IsTicketSupported() || len(helloMsg.GetSessionTicket()) == 0 {
		rrc.cfg.ClientAuth = tls.RequireAndVerifyClientCert
		return rrc.cfg, nil
	}

	return nil, nil
}
