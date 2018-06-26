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

	info = &tls.ClientHelloInfo{
		CipherSuites: []uint16{tls.TLS_RSA_WITH_RC4_128_SHA, 0x1005},
		Conn:         client,
	}

	unusual := tlsl.logUnusualHellos(info)
	assert.True(t, unusual)

	info = &tls.ClientHelloInfo{
		CipherSuites: []uint16{49199, 49200, 49195, 49196, 52392, 52393, 49171, 49161, 49172, 49162, 156, 157, 47, 53, 49170, 10},
		Conn:         client,
	}

	unusual = tlsl.logUnusualHellos(info)
	assert.False(t, unusual)

	info = &tls.ClientHelloInfo{
		CipherSuites: make([]uint16, 0),
		Conn:         client,
	}

	unusual = tlsl.logUnusualHellos(info)
	assert.True(t, unusual)
}
