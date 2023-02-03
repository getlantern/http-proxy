package prefix

import (
	"fmt"
	"net"
	"sync"
	"time"
)

type WritePrefixConn struct {
	net.Conn
	ID string

	prefixBuf []byte
	once      sync.Once
}

func NewWritePrefixConn(ID string, conn net.Conn, prefixBuf []byte) *WritePrefixConn {
	return &WritePrefixConn{
		ID:        ID,
		Conn:      conn,
		prefixBuf: prefixBuf,
	}
}

func (p *WritePrefixConn) Read(b []byte) (n int, err error) {
	return p.Conn.Read(b)
}

func (p *WritePrefixConn) Write(b []byte) (n int, err error) {
	p.once.Do(func() {
		_, err = p.Conn.Write(p.prefixBuf)
		if err == nil {
			log.Debugf("PrefixConn %s (Write): Prefix (%s) written", p.ID, p.prefixBuf)
		}
	})

	// Check the error from the prefix write, if any
	if err != nil {
		return 0, fmt.Errorf(
			"Unable to write prefix (%v): %v", p.prefixBuf, err)
	}

	return p.Conn.Write(b)
}

func (p *WritePrefixConn) Close() error {
	return p.Conn.Close()
}

func (p *WritePrefixConn) LocalAddr() net.Addr {
	return p.Conn.LocalAddr()
}

func (p *WritePrefixConn) RemoteAddr() net.Addr {
	return p.Conn.RemoteAddr()
}

func (p *WritePrefixConn) SetDeadline(t time.Time) error {
	return p.Conn.SetDeadline(t)
}

func (p *WritePrefixConn) SetReadDeadline(t time.Time) error {
	return p.Conn.SetReadDeadline(t)
}

func (p *WritePrefixConn) SetWriteDeadline(t time.Time) error {
	return p.Conn.SetWriteDeadline(t)
}
