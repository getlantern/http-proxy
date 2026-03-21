package broflake

import (
	"context"
	"net"

	"github.com/getlantern/broflake/egress"
	egcmdcommon "github.com/getlantern/broflake/egress/cmd/common"
)

func Wrap(ll net.Listener, certPEM string, keyPEM string) (net.Listener, error) {
	return egress.NewListener(context.Background(), ll, egcmdcommon.GenerateSelfSignedTLSConfig(true))
}
