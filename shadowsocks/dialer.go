package shadowsocks

import (
	"context"
	"net"

	"github.com/Jigsaw-Code/outline-sdk/transport"
)

type LocalDialer struct {
	connections chan net.Conn
}

func (d *LocalDialer) DialStream(ctx context.Context, addr string) (transport.StreamConn, error) {
	cliConn := ctx.Value(ClientConnCtxKey{}).(transport.StreamConn)

	c1, c2 := net.Pipe()
	a := &tcpConnAdapter{c1}
	b := &lfwd{
		Conn:           c2,
		remoteAddr:     cliConn.RemoteAddr(),
		clientTCPConn:  cliConn,
		upstreamTarget: addr,
	}
	d.connections <- b

	return a, nil
}
