package proxy

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestPreferIPV6Dialer(t *testing.T) {
	tests := []struct {
		name     string
		timeout  time.Duration
		network  string
		hostport string
		server   func(t *testing.T, hostport string) func()
		wantErr  bool
	}{
		{
			name:     "IPv6 address",
			timeout:  1 * time.Second,
			network:  "tcp",
			hostport: "[::1]:8080",
			server:   func(t *testing.T, hostport string) func() { return runTestServer(t, "[::1]:8080") },
			wantErr:  false,
		},
		{
			name:     "IPv4 address",
			timeout:  1 * time.Second,
			network:  "tcp",
			hostport: "127.0.0.1:8080",
			server:   func(t *testing.T, hostport string) func() { return runTestServer(t, "127.0.0.1:8080") },
			wantErr:  false,
		},
		{
			name:     "Invalid address",
			timeout:  1 * time.Second,
			network:  "tcp",
			hostport: "invalid",
			server:   nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.server != nil {
				closer := tt.server(t, tt.hostport)
				defer closer()
			}

			dialer := dialWithFastFallback(tt.timeout)
			conn, err := dialer(context.Background(), tt.network, tt.hostport)
			if (err != nil) != tt.wantErr {
				t.Errorf("preferIPV6Dialer() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if conn != nil {
				conn.Close()
			}
		})
	}
}

func runTestServer(t *testing.T, addr string) func() {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()

	return func() {
		listener.Close()
	}
}
