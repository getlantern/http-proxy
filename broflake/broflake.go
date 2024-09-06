package broflake

import (
	"context"
	"net"

	"github.com/getlantern/broflake/egress"
)

func Wrap(ll net.Listener, certPEM string, keyPEM string) (net.Listener, error) {
	// TODO (allan): update the Broflake library to accept cert and key as PEM encoded strings
	// Maybe it's already done. Investigate!
	return egress.NewListener(context.Background(), ll, certPEM, keyPEM)
}
