package replicant

import (
	"net"

	"github.com/getlantern/golog"

	replicant "github.com/OperatorFoundation/shapeshifter-transports/transports/Replicant/v2"
)

var (
	log = golog.LoggerFor("replicant")
)

func Wrap(ll net.Listener, encodedConfig string) (net.Listener, error) {
	serverConfig, err := replicant.DecodeServerConfig(encodedConfig)
	if err != nil {
		return nil, err
	}

	return &ReplicantListener{
		Listener: ll,
		cfg:      serverConfig,
	}, nil
}

type ReplicantListener struct {
	net.Listener

	cfg *replicant.ServerConfig
}

func (l ReplicantListener) Accept() (net.Conn, error) {
	conn, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}

	return replicant.NewServerConnection(conn, *l.cfg)
}
