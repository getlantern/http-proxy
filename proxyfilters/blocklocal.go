package proxyfilters

import (
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/getlantern/iptool"
	"github.com/getlantern/proxy/v3/filters"
)

type resolver interface {
	SplitHostPort(hostport string) (host string, port string, err error)
	ResolveIPAddr(network string, address string) (*net.IPAddr, error)
}

type Resolver struct{}

func (r *Resolver) SplitHostPort(hostport string) (host string, port string, err error) {
	return net.SplitHostPort(hostport)
}
func (r *Resolver) ResolveIPAddr(network string, address string) (*net.IPAddr, error) {
	return net.ResolveIPAddr(network, address)
}

// BlockLocal blocks attempted accesses to localhost unless they're one of the
// listed exceptions.
func BlockLocal(exceptions []string, r resolver) filters.Filter {
	ipt, _ := iptool.New()
	isException := func(host string) bool {
		for _, exception := range exceptions {
			if strings.EqualFold(host, exception) {
				// This is okay, allow it
				return true
			}
		}
		return false
	}

	return filters.FilterFunc(func(cs *filters.ConnectionState, req *http.Request, next filters.Next) (*http.Response, *filters.ConnectionState, error) {
		if isException(req.URL.Host) {
			return next(cs, req)
		}

		host, port, err := r.SplitHostPort(req.URL.Host)
		if err != nil {
			// host didn't have a port, thus splitting didn't work
			host = req.URL.Host
		}

		ipAddr, err := r.ResolveIPAddr("ip", host)

		// If there was an error resolving is probably because it wasn't an address
		// in the form host or host:port
		if err == nil {
			if ipt.IsPrivate(ipAddr) {
				return fail(cs, req, http.StatusForbidden, "%v requested local address %v (%v)", req.RemoteAddr, req.Host, ipAddr)
			}
		}

		// Note: It is important to pass Host as an already resolved and vetted IP in order to avoid
		// DNS rebind attacks should there be any other dialers, that attempt to resolve the host down in the execution path
		addr := ipAddr.String()
		if port != "" && addr != "" {
			req.URL.Host = fmt.Sprintf("%s:%s", addr, port)
		} else if addr != "" {
			req.URL.Host = addr
		}

		return next(cs, req)
	})
}
