package broflake

import (
	"context"
	"net"

	"github.com/getlantern/broflake/egress"
)

func WrapWebSocket(ll net.Listener, certPEM string, keyPEM string) (net.Listener, error) {
	return egress.NewWebSocketListener(context.Background(), ll, certPEM, keyPEM)
}

func WrapWebTransport(addr, certPEM, keyPEM string) (net.Listener, error) {
	return egress.NewWebTransportListener(context.Background(), addr, certPEM, keyPEM)
}
