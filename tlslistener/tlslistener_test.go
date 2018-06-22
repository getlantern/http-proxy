package tlslistener

import (
	"crypto/tls"
	"net"
	"testing"

	"github.com/getlantern/golog"
	"github.com/stretchr/testify/assert"
)

func TestTestEq(t *testing.T) {
	assert.True(t, testEq(nil, nil))
	assert.False(t, testEq(nil, []uint16{1}))
	assert.False(t, testEq([]uint16{1}, nil))
	assert.False(t, testEq([]uint16{1}, []uint16{1, 1}))
	assert.False(t, testEq([]uint16{1}, []uint16{2}))
	assert.True(t, testEq([]uint16{2}, []uint16{2}))
}

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
