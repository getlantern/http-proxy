package broflake

import (
	"context"
	"net"

	"github.com/armon/go-socks5"
	"github.com/getlantern/broflake/common"

	"github.com/getlantern/broflake/egress"
	egcmdcommon "github.com/getlantern/broflake/egress/cmd/common"
)

func Wrap(ll net.Listener, certPEM string, keyPEM string) (net.Listener, error) {
	common.Debugf("@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@")
	common.Debugf("@ DANGER                                                @")
	common.Debugf("@ DANGER                                                @")
	common.Debugf("@ DANGER                                                @")
	common.Debugf("@                                                       @")
	common.Debugf("@ This standalone egress server does not use secure TLS @")
	common.Debugf("@ at the QUIC layer!                                    @")
	common.Debugf("@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@\n")

	// And here's why it doesn't use secure TLS at the QUIC layer
	tlsConfig := egcmdcommon.GenerateSelfSignedTLSConfig(true)

	egressListener, err := egress.NewListener(context.Background(), ll, tlsConfig)
	if err != nil {
		panic(err)
	}
	defer egressListener.Close()

	conf := &socks5.Config{
		Dial:     UoTDialer(),
		Resolver: &UoTResolver{},
	}
	proxy, err := socks5.New(conf)
	if err != nil {
		panic(err)
	}

	common.Debugf("Starting SOCKS5 UoT proxy...")

	err = proxy.Serve(egressListener)
	if err != nil {
		panic(err)
	}
	return egressListener, nil
}
