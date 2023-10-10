package proxy

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/getlantern/errors"
	"github.com/getlantern/netx"
)

const (
	// 5 second DNS resolution timeout is the default on Linux
	resolutionTimeout = 5 * time.Second
)

// Returns a dialer that uses custom DNS servers to resolve the host. It uses all DNS servers
// in parallel and uses the first response it gets.
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
		if ip == nil {
			// the host wasn't an IP, so resolve it
			results := make(chan net.IP, len(resolvers))
			errs := make(chan error, len(resolvers))
			rctx, cancel := context.WithTimeout(ctx, resolutionTimeout)
			defer cancel()
			for _, r := range resolvers {
				resolveInBackground(rctx, r, host, results, errs)
			}
			select {
			case ip = <-results:
				// got a result!
			case <-time.After(resolutionTimeout):
				var resolveErr error
				select {
				case resolveErr = <-errs:
					// got an error
				default:
					// no error, we just timed out
				}
				return nil, errors.New("unable to resolve host %v, last resolution error: %v", host, resolveErr)
			}
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

func resolveInBackground(ctx context.Context, r *net.Resolver, host string, results chan net.IP, errors chan error) {
	go func() {
		ips, err := r.LookupIPAddr(ctx, host)
		if err != nil {
			errors <- err
			return
		}
		if len(ips) > 0 {
			// Google anomaly detection can be triggered very often over IPv6.
			// Prefer IPv4 to mitigate, see issue #97
			// If no IPv4 is available, fall back to IPv6
			for _, candidate := range ips {
				if candidate.IP.To4() != nil {
					results <- candidate.IP
					return
				}
			}
			// We couldn't find an IPv4, so just use the first one (at this point we assume it's IPv6)
			results <- ips[0].IP
		}
	}()
}
