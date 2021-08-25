package tlsmasq

import (
	"context"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"net"

	"github.com/getlantern/tlsmasq"
	"github.com/getlantern/tlsmasq/ptlshs"
)

func Wrap(ll net.Listener, certFile string, keyFile string, originAddr string, secret string,
	tlsMinVersion uint16, tlsCipherSuites []uint16, onNonFatalErrors func(error)) (net.Listener, error) {

	var secretBytes ptlshs.Secret
	_secretBytes, decodeErr := hex.DecodeString(secret)
	if decodeErr != nil {
		return nil, fmt.Errorf(`unable to decode secret string "%v": %v`, secret, decodeErr)
	}
	if copy(secretBytes[:], _secretBytes) != 52 {
		return nil, fmt.Errorf(`secret string did not parse to 52 bytes: "%v"`, secret)
	}

	cert, keyErr := tls.LoadX509KeyPair(certFile, keyFile)
	if keyErr != nil {
		return nil, fmt.Errorf("unable to load key file for tlsmasq: %v", keyErr)
	}

	dialOrigin := func(ctx context.Context) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "tcp", originAddr)
	}

	nonFatalErrChan := make(chan error)
	go func() {
		for err := range nonFatalErrChan {
			onNonFatalErrors(err)
		}
	}()

	listenerCfg := tlsmasq.ListenerConfig{
		ProxiedHandshakeConfig: ptlshs.ListenerConfig{
			DialOrigin: dialOrigin,
			Secret:     secretBytes,
		},
		TLSConfig: &tls.Config{
			Certificates: []tls.Certificate{cert},
			MinVersion:   tlsMinVersion,
			CipherSuites: tlsCipherSuites,
		},
	}

	return tlsmasq.WrapListener(ll, listenerCfg), nil
}
