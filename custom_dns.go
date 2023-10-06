package proxy

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/getlantern/errors"
	"github.com/getlantern/netx"
)

// Returns a dialer that uses custom DNS servers to resolve the host.
func customDNSDialer(dnsServers []string, timeout time.Duration) (func(context.Context, string, string) (net.Conn, error), error) {
	resolvers := make([]*net.Resolver, 0, len(dnsServers))
	if len(dnsServers) == 0 {
		log.Debug("Will resolve DNS using system DNS servers")
		resolvers = append(resolvers, net.DefaultResolver)
	} else {
		log.Debugf("Will resolve DNS using %v", dnsServers)
		for _, _dnsServer := range dnsServers {
			dnsServer := _dnsServer
			r := &net.Resolver{
				PreferGo: true,
				Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
					return netx.DialContext(ctx, "udp", dnsServer)
				},
			}
			resolvers = append(resolvers, r)
		}
	}

	dial := func(ctx context.Context, network, addr string) (net.Conn, error) {
		// resolve separately so that we can track the DNS resolution time
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, errors.New("invalid address %v: %v", addr, err)
		}
		ip := net.ParseIP(host)
		var resolveErr error
		if ip == nil {
			// the host wasn't an IP, so resolve it
		resolveLoop:
			for _, r := range resolvers {
				var ips []net.IPAddr
				// Note - 5 seconds is the default Linux DNS timeout
				rctx, cancel := context.WithTimeout(ctx, 5*time.Second)
				ips, resolveErr = r.LookupIPAddr(rctx, host)
				cancel()
				if resolveErr == nil && len(ips) > 0 {
					// Google anomaly detection can be triggered very often over IPv6.
					// Prefer IPv4 to mitigate, see issue #97
					// If no IPv4 is available, fall back to IPv6
					for _, candidate := range ips {
						if candidate.IP.To4() != nil {
							ip = candidate.IP
							break resolveLoop
						}
					}
					// We couldn't find an IPv4, so just use the first one (at this point we assume it's IPv6)
					ip = ips[0].IP
					break resolveLoop
				}
			}
		}
		if ip == nil {
			return nil, errors.New("unable to resolve host %v, last resolution error: %v", host, resolveErr)
		}

		resolvedAddr := fmt.Sprintf("%s:%s", ip, port)
		d := &net.Dialer{
			Deadline: time.Now().Add(timeout),
		}
		conn, dialErr := d.DialContext(ctx, "tcp", resolvedAddr)
		if dialErr != nil {
			return nil, dialErr
		}

		return conn, nil
	}

	return dial, nil
}
