package proxyfilters

import (
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/getlantern/iptool"
	"github.com/getlantern/proxy/v3/filters"
)

// BlockLocal blocks attempted accesses to localhost unless they're one of the
// listed exceptions.
func BlockLocal(exceptions []string) filters.Filter {
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

		host, port, err := net.SplitHostPort(req.URL.Host)
		if err != nil {
			// host didn't have a port, thus splitting didn't work
			host = req.URL.Host
		}

		ipAddr, err := net.ResolveIPAddr("ip", host)

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
