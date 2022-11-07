/*
	MIT License

	Copyright (c) 2020 Operator Foundation

	Permission is hereby granted, free of charge, to any person obtaining a copy
	of this software and associated documentation files (the "Software"), to deal
	in the Software without restriction, including without limitation the rights
	to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
	copies of the Software, and to permit persons to whom the Software is
	furnished to do so, subject to the following conditions:

	The above copyright notice and this permission notice shall be included in all
	copies or substantial portions of the Software.

	THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
	IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
	FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
	AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
	LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
	OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
	SOFTWARE.
*/

package replicant

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

// Create outgoing transport connection
func (config ClientConfig) Dial() (net.Conn, error) {
	// Verify the transport name on the config
	if config.Transport != "Replicant" {
		return nil, errors.New("incorrect transport name")
	}

	dialTimeout := time.Minute * 5
	conn, dialErr := net.DialTimeout("tcp", config.ServerAddress, dialTimeout)
	if dialErr != nil {
		return nil, dialErr
	}

	transportConn, err := NewClientConnection(conn, config)
	if err != nil {
		if conn != nil {
			_ = conn.Close()
		}
		return nil, err
	}

	return transportConn, nil
}

// Create listener for incoming transport connection
func (config ServerConfig) Listen() (net.Listener, error) {
	// Verify the transport name on the config
	if config.Transport != "Replicant" {
		return nil, errors.New("incorrect transport name")
	}

	addr, resolveErr := ResolveAddr(config.ServerAddress)
	if resolveErr != nil {
		return nil, resolveErr
	}

	ln, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return nil, err
	}

	return newReplicantTransportListener(ln, config), nil
}

func (listener *replicantTransportListener) Addr() net.Addr {
	interfaces, _ := net.Interfaces()
	addrs, _ := interfaces[0].Addrs()
	return addrs[0]
}

// Accept waits for and returns the next connection to the listener.
func (listener *replicantTransportListener) Accept() (net.Conn, error) {
	conn, err := listener.listener.Accept()
	if err != nil {
		return nil, err
	}
	if conn == nil {
		return nil, errors.New("tcp connection is nil")
	}

	config := listener.config

	newServerConn, serverConnError := NewServerConnection(conn, config)
	if serverConnError != nil {
		conn.Close()
		return nil, serverConnError
	}
	if newServerConn == nil {
		conn.Close()
		return nil, errors.New("newServerConn is nil")
	}

	return newServerConn, nil
}

// Close closes the transport listener.
// Any blocked Accept operations will be unblocked and return errors.
func (listener *replicantTransportListener) Close() error {
	return listener.listener.Close()
}

var _ net.Listener = (*replicantTransportListener)(nil)

// Resolve an address string into a net.TCPAddr. We are a bit more strict than
// net.ResolveTCPAddr; we don't allow an empty host or port, and the host part
// must be a literal IP address.
func ResolveAddr(addrStr string) (*net.TCPAddr, error) {
	ipStr, portStr, err := net.SplitHostPort(addrStr)
	if err != nil {
		// Before the fixing of bug #7011, tor doesn't put brackets around IPv6
		// addresses. Split after the last colon, assuming it is a port
		// separator, and try adding the brackets.
		parts := strings.Split(addrStr, ":")
		if len(parts) <= 2 {
			return nil, err
		}
		addrStr := "[" + strings.Join(parts[:len(parts)-1], ":") + "]:" + parts[len(parts)-1]
		ipStr, portStr, err = net.SplitHostPort(addrStr)
	}
	if err != nil {
		return nil, err
	}
	if ipStr == "" {
		return nil, net.InvalidAddrError(fmt.Sprintf("address string %q lacks a host part", addrStr))
	}
	if portStr == "" {
		return nil, net.InvalidAddrError(fmt.Sprintf("address string %q lacks a port part", addrStr))
	}
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return nil, net.InvalidAddrError(fmt.Sprintf("not an IP string: %q", ipStr))
	}
	port, err := parsePort(portStr)
	if err != nil {
		return nil, err
	}
	return &net.TCPAddr{IP: ip, Port: port}, nil
}

func parsePort(portStr string) (int, error) {
	port, err := strconv.ParseUint(portStr, 10, 16)
	return int(port), err
}
