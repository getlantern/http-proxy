// Package tlslistener provides a wrapper around tls.Listen that allows
// descending into the wrapped net.Conn
package tlslistener

import (
	"crypto/tls"
	"net"
	"strconv"

	"github.com/getlantern/golog"
	"github.com/getlantern/tlsdefaults"
)

// Wrap wraps the specified listener in our default TLS listener.
func Wrap(wrapped net.Listener, keyFile string, certFile string) (net.Listener, error) {
	cfg, err := tlsdefaults.BuildListenerConfig(wrapped.Addr().String(), keyFile, certFile)
	if err != nil {
		return nil, err
	}

	listener := &tlslistener{wrapped, cfg, golog.LoggerFor("lantern-proxy-tlslistener")}
	cfg.GetConfigForClient = listener.debugClientHello
	return listener, nil
}

type tlslistener struct {
	wrapped net.Listener
	cfg     *tls.Config
	log     golog.Logger
}

func (l *tlslistener) Accept() (net.Conn, error) {
	conn, err := l.wrapped.Accept()
	if err != nil {
		return nil, err
	}
	return &tlsconn{tls.Server(conn, l.cfg), conn}, nil
}

func (l *tlslistener) Addr() net.Addr {
	return l.wrapped.Addr()
}

func (l *tlslistener) Close() error {
	return l.wrapped.Close()
}

// These are the standard suites Lantern clients typically report, and
// typically in the same order. While we have yet to confirm it, it appears
// likely the second is mobile and the first is desktop.
var standardSuites = [][]uint16{
	[]uint16{49199, 49200, 49195, 49196, 52392, 52393, 49171, 49161, 49172, 49162, 156, 157, 47, 53, 49170, 10},
	[]uint16{52392, 52393, 49199, 49200, 49195, 49196, 49171, 49161, 49172, 49162, 156, 157, 47, 53, 49170, 10},
	[]uint16{49199, 49195, 49200, 49196, 49171, 49161, 49172, 49162, 156, 157, 47, 53, 49170, 10},
}

func (l *tlslistener) debugClientHello(info *tls.ClientHelloInfo) (*tls.Config, error) {
	l.logUnusualHellos(info)

	// Returning nil just tells the caller to use the standard config.
	return nil, nil
}

// logUnusualHellos logs if a client hello contains unusual cipher suites.
// If it's unusual, this returns true.
func (l *tlslistener) logUnusualHellos(info *tls.ClientHelloInfo) bool {
	if len(info.CipherSuites) == 0 {
		l.log.Errorf("Client Hello has no cipher suites %v", info.Conn.RemoteAddr())
		return true
	}
	for _, suite := range standardSuites {
		if testEq(suite, info.CipherSuites) {
			return false
		}
	}
	l.log.Debugf("Unexpected suites from client %v: %v, %v", info.Conn.RemoteAddr(), info.CipherSuites, l.suiteStrings(info))
	return true
}

func testEq(a, b []uint16) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

func (l *tlslistener) suiteStrings(info *tls.ClientHelloInfo) []string {
	ints := info.CipherSuites
	strs := make([]string, len(ints))
	for index, i := range ints {
		str, ok := suites[i]
		if ok {
			strs[index] = str
		} else {
			strs[index] = strconv.Itoa(int(i))
		}
	}
	return strs
}

var suites = map[uint16]string{
	tls.TLS_RSA_WITH_RC4_128_SHA:                "TLS_RSA_WITH_RC4_128_SHA",
	tls.TLS_RSA_WITH_3DES_EDE_CBC_SHA:           "TLS_RSA_WITH_3DES_EDE_CBC_SHA",
	tls.TLS_RSA_WITH_AES_128_CBC_SHA:            "TLS_RSA_WITH_AES_128_CBC_SHA",
	tls.TLS_RSA_WITH_AES_256_CBC_SHA:            "TLS_RSA_WITH_AES_256_CBC_SHA",
	tls.TLS_RSA_WITH_AES_128_CBC_SHA256:         "TLS_RSA_WITH_AES_128_CBC_SHA256",
	tls.TLS_RSA_WITH_AES_128_GCM_SHA256:         "TLS_RSA_WITH_AES_128_GCM_SHA256",
	tls.TLS_RSA_WITH_AES_256_GCM_SHA384:         "TLS_RSA_WITH_AES_256_GCM_SHA384",
	tls.TLS_ECDHE_ECDSA_WITH_RC4_128_SHA:        "TLS_ECDHE_ECDSA_WITH_RC4_128_SHA",
	tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA:    "TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA",
	tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA:    "TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA",
	tls.TLS_ECDHE_RSA_WITH_RC4_128_SHA:          "TLS_ECDHE_RSA_WITH_RC4_128_SHA",
	tls.TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA:     "TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA",
	tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA:      "TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA",
	tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA:      "TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA",
	tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256: "TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256",
	tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256:   "TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256",
	tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256:   "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256",
	tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256: "TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256",
	tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384:   "TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384",
	tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384: "TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384",
	tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305:    "TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305",
	tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305:  "TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305",
}

type tlsconn struct {
	net.Conn
	wrapped net.Conn
}

func (conn *tlsconn) Wrapped() net.Conn {
	return conn.wrapped
}
