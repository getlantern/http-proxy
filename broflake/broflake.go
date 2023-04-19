package broflake

import (
	"context"
	"net"

	"github.com/getlantern/broflake/egress"
)

func Wrap(ll net.Listener, certFile string, keyFile string) (net.Listener, error) {
	return egress.NewListener(context.Background(), ll, certFile, keyFile)
}
