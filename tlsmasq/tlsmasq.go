package tlsmasq

import (
	"crypto/tls"
	"fmt"
	"net"

	"github.com/getlantern/tlsmasq"
	"github.com/getlantern/tlsmasq/ptlshs"
)

func Wrap(ll net.Listener, certFile string, keyFile string, proxiedAddr string, secret [52]byte) (net.Listener, error) {
	cert, keyErr := tls.LoadX509KeyPair(certFile, keyFile)
	if keyErr != nil {
		return nil, fmt.Errorf("Unable to load key file for tlsmasq: %v", keyErr)
	}

	dialOrigin := func() (net.Conn, error) { return net.Dial("tcp", proxiedAddr) }

	tlsConfig := &tls.Config{Certificates: []tls.Certificate{cert}}
	listenerCfg := tlsmasq.ListenerConfig{
		ProxiedHandshakeConfig: ptlshs.ListenerConfig{
			DialOrigin: dialOrigin,
			Secret:     secret,
		},
		TLSConfig: tlsConfig,
	}

	return tlsmasq.WrapListener(ll, listenerCfg), nil
}
