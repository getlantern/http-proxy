package shadowsocks

import (
	"context"
	"fmt"
	"net"

	"github.com/Jigsaw-Code/outline-sdk/transport"
)

type LocalDialer struct {
	connections chan net.Conn
}

func (d *LocalDialer) DialStream(ctx context.Context, addr string) (transport.StreamConn, error) {
	cliConn, ok := ctx.Value(clientConnCtxKey{}).(transport.StreamConn)
	if !ok {
		return nil, fmt.Errorf("expected stream connection in context but received type %T", ctx.Value(clientConnCtxKey{}))
	}

	// We create a pair of connection pipes, where one end is returned to the function caller (you can see more details here: https://github.com/Jigsaw-Code/outline-ss-server/blob/v1.5.0/service/tcp.go#L293-L299)
	// and another is sent to the listener connection channel. The connection channel is used to effectively proxy the connection data to the client.
	// So whenever we receive a shadowsocks connection:
	// (1) the connection it's received by the local listener;
	// (2) the connection passes through the shadowsocks authenticator that validate/decrypt the connection with the ciphers we provided and;
	// (3) we finally send the connection to the connection channel (proxying the connection).
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
