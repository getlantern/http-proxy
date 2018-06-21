package tlslistener

import (
	"crypto/tls"
	"net"
	"testing"

	"github.com/getlantern/golog"
)

func TestWrap(t *testing.T) {

	tlsl := &tlslistener{
		cfg: &tls.Config{},
		log: golog.LoggerFor("tlslistener-test"),
	}

	_, client := net.Pipe()
	info := &tls.ClientHelloInfo{
		CipherSuites: []uint16{tls.TLS_RSA_WITH_RC4_128_SHA, 0x1005},
		Conn:         client,
	}

	tlsl.debugClientHello(info)
}
