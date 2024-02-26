package proxy

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"time"
)

func preferIPV6Dialer(timeout time.Duration) func(ctx context.Context, network, hostport string) (net.Conn, error) {
	return func(ctx context.Context, network, hostport string) (net.Conn, error) {
		tcpAddr, err := resolveAddressPreferIPv6(hostport)
		if err != nil {
			return nil, err
		}

		dialer := net.Dialer{
			Deadline: time.Now().Add(timeout),
		}
		conn, err := dialer.DialContext(ctx, "tcp6", tcpAddr.String())
		if err != nil {
			var e *net.AddrError
			// if this is a network address error, we will retry with the specified network instead (tcp4 most likely)
			if errors.As(err, &e) {
				conn, err = dialer.DialContext(ctx, network, hostport)
			}
		}
		return conn, err
	}
}

func resolveAddressPreferIPv6(hostport string) (*net.TCPAddr, error) {
	host, portStr, err := net.SplitHostPort(hostport)
	if err != nil {
		//include error message in the return
		return nil, fmt.Errorf("unable to split host and port: %w", err)
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, fmt.Errorf("unable to parse port: %w", err)
	}

	// Attempt to directly resolve as IPv6 to avoid unnecessary lookups
	ipv6Addr, err := net.ResolveIPAddr("ip6", host)
	if err == nil && ipv6Addr.IP.To4() == nil {
		return &net.TCPAddr{IP: ipv6Addr.IP, Port: port}, nil
	}

	// If IPv6 resolution failed, fall back to a full lookup and prefer any IPv6 addresses found
	ips, err := net.LookupIP(host)
	if err != nil {
		return nil, fmt.Errorf("unable to resolve IP addresses for host: %w", err)
	}

	for _, ip := range ips {
		if ip.To4() == nil {
			return &net.TCPAddr{IP: ip, Port: port}, nil
		}
	}

	// If no IPv6 addresses are found, try resolving as IPv4
	ipv4Addr, err := net.ResolveTCPAddr("tcp4", hostport)
	if err == nil {
		return ipv4Addr, nil
	}

	return nil, fmt.Errorf("unable to resolve any IP addresses for host: %s", host)
}
