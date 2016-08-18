// Package kcplistener provides a listener using TLS over KCP (see
// https://github.com/xtaci/kcp-go).
package kcplistener

import (
	"net"

	"github.com/getlantern/golog"
	"github.com/getlantern/tlsdefaults"
	"github.com/xtaci/kcp-go"
)

var (
	log = golog.LoggerFor("kcplistener")
)

// NewListener creates a new KCP listener that listens at the given Address
func NewListener(addr string, pkfile string, certfile string) (net.Listener, error) {
	// Right now we're just hardcoding the data and parity shards for the error
	// correcting codes. See https://github.com/klauspost/reedsolomon#usage for
	// a discussion of these.
	l, err := kcp.ListenWithOptions(addr, nil, 10, 3)
	if err != nil {
		return nil, err
	}
	tl, err := tlsdefaults.NewListener(&kcplistener{l}, pkfile, certfile)
	if err != nil {
		l.Close()
		return nil, err
	}
	return tl, nil
}

type kcplistener struct {
	wrapped *kcp.Listener
}

func (l *kcplistener) Accept() (net.Conn, error) {
	return l.wrapped.Accept()
}

func (l *kcplistener) Addr() net.Addr {
	return l.wrapped.Addr()
}

func (l *kcplistener) Close() error {
	return l.wrapped.Close()
}
