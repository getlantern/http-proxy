package configserverfilter

import (
	"crypto/tls"
	"net"
)

type Dial func(network, address string) (net.Conn, error)

func Dialer(d Dial, opts *Options) Dial {
	return func(network, address string) (net.Conn, error) {
		if in(address, opts.Domains) {
			log.Debugf("Use TLS dialer for %s", address)
			return tls.Dial(network, address, nil)
		}
		return d(network, address)
	}
}
