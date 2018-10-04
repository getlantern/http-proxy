package configserverfilter

import (
	"context"
	"crypto/tls"
	"net"
)

type Dial func(ctx context.Context, network, address string) (net.Conn, error)

func Dialer(d Dial, opts *Options) Dial {
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		conn, err := d(ctx, network, address)
		if err != nil {
			return conn, err
		}
		if matched := in(address, opts.Domains); matched != "" {
			conn = tls.Client(conn, &tls.Config{ServerName: matched})
			log.Debugf("Added TLS to connection to %s", address)
		}
		return conn, err
	}
}
