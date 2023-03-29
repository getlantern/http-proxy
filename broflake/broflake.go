package broflake

import (
	"context"
	"net"

	"github.com/getlantern/broflake/egress"
)

func Wrap(ll net.Listener) (net.Listener, error) {
	return egress.NewListener(context.Background(), ll)
}
