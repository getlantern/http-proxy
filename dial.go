package proxy

import (
	"context"
	"net"
	"time"
)

func dialWithFastFallback(timeout time.Duration) func(ctx context.Context, network, hostport string) (net.Conn, error) {
	return func(ctx context.Context, network, hostport string) (net.Conn, error) {
		dialer := &net.Dialer{
			Timeout: timeout,
		}
		return dialer.DialContext(ctx, network, hostport)
	}
}
