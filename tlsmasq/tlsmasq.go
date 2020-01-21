package tlsmasq

import (
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"net"

	"github.com/getlantern/tlsmasq"
	"github.com/getlantern/tlsmasq/ptlshs"
)

func Wrap(ll net.Listener, certFile string, keyFile string, originAddr string, secret string, onNonFatalErrors func(error)) (net.Listener, error) {
	var secretBytes ptlshs.Secret
	if _secretBytes, decodeErr := hex.DecodeString(secret); decodeErr != nil {
		return nil, fmt.Errorf(`unable to decode secret string "%v": %v`, secret, decodeErr)
	} else {
		if copy(secretBytes[:], _secretBytes) != 52 {
			return nil, fmt.Errorf(`secret string did not parse to 52 bytes: "%v"`, secret)
		}
	}

	cert, keyErr := tls.LoadX509KeyPair(certFile, keyFile)
	if keyErr != nil {
		return nil, fmt.Errorf("unable to load key file for tlsmasq: %v", keyErr)
	}

	dialOrigin := func() (net.Conn, error) { return net.Dial("tcp", originAddr) }

	nonFatalErrChan := make(chan error)
	go func() {
		for err := range nonFatalErrChan {
			onNonFatalErrors(err)
		}
	}()

	tlsConfig := &tls.Config{Certificates: []tls.Certificate{cert}}
	listenerCfg := tlsmasq.ListenerConfig{
		ProxiedHandshakeConfig: ptlshs.ListenerConfig{
			DialOrigin: dialOrigin,
			Secret:     secretBytes,
		},
		TLSConfig: tlsConfig,
	}

	return tlsmasq.WrapListener(ll, listenerCfg), nil
}
