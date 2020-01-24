package lampshade

import (
	"crypto/rsa"
	"crypto/tls"
	"fmt"
	"net"
	"time"

	"github.com/getlantern/http-proxy/buffers"
	"github.com/getlantern/lampshade"
)

func Wrap(ll net.Listener, certFile string, keyFile string, keyCacheSize int, maxClientInitAge time.Duration, onListenerError func(net.Conn, error)) (net.Listener, error) {
	cert, keyErr := tls.LoadX509KeyPair(certFile, keyFile)
	if keyErr != nil {
		return nil, fmt.Errorf("Unable to load key file for lampshade: %v", keyErr)
	}
	return lampshade.WrapListener(
		ll,
		buffers.Pool(),
		cert.PrivateKey.(*rsa.PrivateKey),
		&lampshade.ListenerOpts{
			AckOnFirst:       true,
			KeyCacheSize:     keyCacheSize,
			MaxClientInitAge: maxClientInitAge,
			OnError:          onListenerError}), nil
}
