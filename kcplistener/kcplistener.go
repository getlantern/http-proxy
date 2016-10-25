// Package kcplistener provides a listener using TLS over KCP (see
// https://github.com/xtaci/kcp-go).
package kcplistener

import (
	"net"

	"github.com/getlantern/cmux"
	"github.com/getlantern/golog"
	"github.com/getlantern/snappyconn"
	"github.com/xtaci/kcp-go"
)

var (
	log = golog.LoggerFor("kcplistener")
)

// NewListener creates a new KCP listener that listens at the given Address
func NewListener(addr string) (net.Listener, error) {
	// Right now we're just hardcoding the data and parity shards for the error
	// correcting codes. See https://github.com/klauspost/reedsolomon#usage for
	// a discussion of these.
	block, _ := kcp.NewNoneBlockCrypt(nil)
	l, err := kcp.ListenWithOptions(addr, block, 10, 3)
	if err != nil {
		return nil, err
	}
	l.SetDSCP(0)
	l.SetReadBuffer(4194304)
	l.SetWriteBuffer(4194304)
	return cmux.Listen(&cmux.ListenOpts{Listener: &kcplistener{l}}), nil
}

type kcplistener struct {
	wrapped *kcp.Listener
}

func (l *kcplistener) Accept() (net.Conn, error) {
	conn, err := l.wrapped.AcceptKCP()
	if err != nil {
		return nil, err
	}
	applyDefaultConnParameters(conn)
	return snappyconn.Wrap(conn), err
}

// applyDefaultConnParameters applies the defaults used in kcptun
// See https://github.com/xtaci/kcptun/blob/75923fb08f3bd67acbc212f6b6aac0a445decf72/client/main.go#L276
func applyDefaultConnParameters(conn *kcp.UDPSession) {
	conn.SetStreamMode(true)
	conn.SetNoDelay(0, 20, 2, 1)
	conn.SetWindowSize(128, 1024)
	conn.SetMtu(1350)
	conn.SetACKNoDelay(false)
	conn.SetKeepAlive(10)
}

func (l *kcplistener) Addr() net.Addr {
	return l.wrapped.Addr()
}

func (l *kcplistener) Close() error {
	return l.wrapped.Close()
}
