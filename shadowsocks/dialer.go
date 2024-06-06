package shadowsocks

import (
	"context"
	"net"

	"github.com/Jigsaw-Code/outline-sdk/transport"
)

type LocalDialer struct {
	Dialer net.Dialer
}

func (d *LocalDialer) DialStream(ctx context.Context, addr string) (transport.StreamConn, error) {
	log.Debugf("Dialing %s locally", addr)
	conn, err := d.Dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, err
	}
	return conn.(*net.TCPConn), nil
}
