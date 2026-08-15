package proxyfilters

import (
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/getlantern/proxy/v3/filters"
)

type resolver interface {
	ResolveIPAddr(network string, address string) (*net.IPAddr, error)
}

type Resolver struct{}

func (r *Resolver) ResolveIPAddr(network string, address string) (*net.IPAddr, error) {
	return net.ResolveIPAddr(network, address)
}

// BlockLocal blocks attempted accesses to localhost unless they're one of the
// listed exceptions.
func BlockLocal(exceptions []string, r resolver) filters.Filter {
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
		targetHost := req.URL.Host
		if targetHost == "" {
			// Origin-form requests carry the authority in Request.Host. This is
			// how legacy clients send HTTP requests over persistent proxy tunnels.
			targetHost = req.Host
		}

		host, port, err := net.SplitHostPort(targetHost)
		if err != nil {
			// host didn't have a port, thus splitting didn't work
			host = targetHost
		}

		// Check the bare host too, so a hostname exception matches with
		// or without the default port.
		if isException(targetHost) || isException(host) {
			return next(cs, req)
		}

		ipAddr, err := r.ResolveIPAddr("ip", host)

		// If there was an error resolving is probably because it wasn't an address
		// in the form host or host:port
		if err == nil {
			if ipAddr.IP.IsPrivate() || !ipAddr.IP.IsGlobalUnicast() {
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
