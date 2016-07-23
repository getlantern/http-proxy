package proxy

import (
	"errors"
	"net"
	"strconv"
	"time"
)

// preferIPV4Dialer returns a function with same signature as net.Dial, but
// always dials the host to its IPv4 address, unless it's already in IP address
// form.
func preferIPV4Dialer(timeout time.Duration) func(network, hostport string) (net.Conn, error) {
	return func(network, hostport string) (net.Conn, error) {
		tcpAddr, err := tcpAddrPrefer4(hostport)
		if err != nil {
			return nil, err
		}
		chResult := make(chan struct {
			conn net.Conn
			err  error
		})
		go func() {
			conn, err := net.DialTCP("tcp4", nil, tcpAddr)
			select {
			case chResult <- struct {
				conn net.Conn
				err  error
			}{conn, err}:
			default: // no receiver
				if conn != nil {
					_ = conn.Close()
				}
			}
		}()
		t := time.NewTimer(timeout)
		select {
		case <-t.C:
			return nil, errors.New("Dial timeout")
		case res := <-chResult:
			return res.conn, res.err
		}
	}
}

func tcpAddrPrefer4(hostport string) (*net.TCPAddr, error) {
	host, port, err := net.SplitHostPort(hostport)
	if err != nil {
		return nil, err
	}
	ip := net.ParseIP(host)
	if ip != nil {
		// if host is in IP address form, use it as is
		p, err := strconv.Atoi(port)
		if err != nil {
			return nil, err
		}
		return &net.TCPAddr{ip, p, ""}, nil
	}
	return net.ResolveTCPAddr("tcp4", hostport)
}
