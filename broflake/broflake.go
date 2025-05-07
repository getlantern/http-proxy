package broflake

import (
	"context"
	"net"

	"github.com/getlantern/broflake/egress"
)

func Wrap(ll net.Listener, certPEM string, keyPEM string) (net.Listener, error) {
	return egress.NewWebSocketListener(context.Background(), ll, certPEM, keyPEM)
}
