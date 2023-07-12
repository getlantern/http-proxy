package broflake

import (
	"context"
	"net"

	"github.com/getlantern/broflake/egress"
)

// WrapWebSocket wraps the given listener with a WebSocket listener from broflake.
func WrapWebSocket(ll net.Listener, certPEM string, keyPEM string) (net.Listener, error) {
	return egress.NewWebSocketListener(context.Background(), ll, certPEM, keyPEM)
}

// NewWebTransportListener creates a new listener for WebTransport connections.
func NewWebTransportListener(addr, certPEM, keyPEM string) (net.Listener, error) {
	return egress.NewWebTransportListener(context.Background(), addr, certPEM, keyPEM)
}
