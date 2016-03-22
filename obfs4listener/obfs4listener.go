package obfs4listener

import (
	"fmt"
	"net"
	"os"

	"github.com/Yawning/obfs4/transports/obfs4"

	"git.torproject.org/pluggable-transports/goptlib.git"
	"git.torproject.org/pluggable-transports/obfs4.git/transports/base"
)

func NewListener(addr string, stateDir string) (net.Listener, error) {
	err := os.MkdirAll(stateDir, 0700)
	if err != nil {
		return nil, fmt.Errorf("Unable to make statedir at %v: %v", stateDir, err)
	}

	tr := &obfs4.Transport{}
	sf, err := tr.ServerFactory(stateDir, &pt.Args{})
	if err != nil {
		return nil, fmt.Errorf("Unable to create obfs4 server factory: %v", err)
	}
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("Unable to listen at %v: %v", addr, err)
	}

	return &obfs4listener{l, sf}, nil
}

type obfs4listener struct {
	wrapped net.Listener
	sf      base.ServerFactory
}

func (l *obfs4listener) Accept() (net.Conn, error) {
	conn, err := l.wrapped.Accept()
	if err != nil {
		return nil, err
	}
	return l.sf.WrapConn(conn)
}

func (l *obfs4listener) Addr() net.Addr {
	return l.wrapped.Addr()
}

func (l *obfs4listener) Close() error {
	return l.wrapped.Close()
}
