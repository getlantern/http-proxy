package shadowsocks

import (
	"context"
	"net"

	"github.com/Jigsaw-Code/outline-sdk/transport"
	"github.com/Jigsaw-Code/outline-ss-server/service/metrics"
)

type LocalDialer struct {
	connections chan net.Conn
}

func (d *LocalDialer) DialStream(ctx context.Context, addr string) (transport.StreamConn, error) {
	cliConn := ctx.Value(ClientConnCtxKey{}).(transport.StreamConn)

	c1, c2 := net.Pipe()
	bytesSent := int64(0)
	bytesReceived := int64(0)
	a := metrics.MeasureConn(&tcpConnAdapter{c1}, &bytesSent, &bytesReceived)
	b := &lfwd{
		Conn:           c2,
		remoteAddr:     cliConn.RemoteAddr(),
		clientTCPConn:  cliConn,
		upstreamTarget: addr,
	}
	d.connections <- b

	return a, nil
}
