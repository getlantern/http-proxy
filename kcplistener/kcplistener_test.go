package kcplistener

import (
	"io"
	"net"
	"testing"

	"github.com/getlantern/cmux"
	"github.com/getlantern/snappyconn"
	"github.com/stretchr/testify/assert"
	"github.com/xtaci/kcp-go"
)

func TestRoundTrip(t *testing.T) {
	l, err := NewListener("localhost:0")
	if !assert.NoError(t, err, "Unable to create listener") {
		return
	}
	defer l.Close()

	go func() {
		for {
			conn, err := l.Accept()
			if err == nil {
				// Echo
				io.Copy(conn, conn)
			}
		}
	}()

	b := []byte("Hi There")

	block, _ := kcp.NewNoneBlockCrypt(nil)
	dialer := cmux.Dialer(&cmux.DialerOpts{
		Dial: func(network, addr string) (net.Conn, error) {
			conn, err := kcp.DialWithOptions(addr, block, 10, 3)
			if err != nil {
				return nil, err
			}
			applyDefaultConnParameters(conn)
			conn.SetDSCP(0)
			conn.SetReadBuffer(4194304)
			conn.SetWriteBuffer(4194304)
			return snappyconn.Wrap(conn), nil
		},
	})

	conn, err := dialer("tcp", l.Addr().String())
	if !assert.NoError(t, err, "Unable to dial good conn") {
		return
	}

	_, err = conn.Write(b)
	if !assert.NoError(t, err, "Unable to write") {
		return
	}
	e := make([]byte, len(b))
	_, err = conn.Read(e)
	if !assert.NoError(t, err, "Unable to read") {
		return
	}
	assert.Equal(t, string(b), string(e), "Echoed did not match written")
}
