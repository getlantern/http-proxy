package kcplistener

import (
	"crypto/tls"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xtaci/kcp-go"
)

func TestRoundTrip(t *testing.T) {
	pkfile := "pk.pem"
	certfile := "cert.pem"
	l, err := NewListener("localhost:0", pkfile, certfile)
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
	_conn, err := kcp.DialWithOptions(l.Addr().String(), block, 10, 3)
	if !assert.NoError(t, err, "Unable to dial good conn") {
		return
	}
	_conn.SetStreamMode(true)
	_conn.SetNoDelay(0, 20, 2, 1)
	_conn.SetWindowSize(128, 1024)
	_conn.SetMtu(1350)
	_conn.SetACKNoDelay(false)
	_conn.SetKeepAlive(10)
	_conn.SetDSCP(0)
	_conn.SetReadBuffer(4194304)
	_conn.SetWriteBuffer(4194304)
	conn := tls.Client(_conn, &tls.Config{
		InsecureSkipVerify: true,
	})
	defer conn.Close()

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
