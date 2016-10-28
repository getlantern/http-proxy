package kcplistener

import (
	"io"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/getlantern/cmux"
	"github.com/getlantern/idletiming"
	"github.com/getlantern/snappyconn"
	"github.com/stretchr/testify/assert"
	"github.com/xtaci/kcp-go"
)

var (
	idleTimeout = 250 * time.Millisecond
)

func TestRoundTrip(t *testing.T) {
	l, err := NewListener("localhost:0")
	if !assert.NoError(t, err, "Unable to create listener") {
		return
	}
	defer l.Close()

	openConns := int32(0)
	go func() {
		for {
			conn, err := l.Accept()
			if err == nil {
				log.Debug("Accepted")
				atomic.AddInt32(&openConns, 1)
				conn = idletiming.Conn(conn, idleTimeout, func() {
					log.Debug("Conn closed")
				})
				// Echo
				io.Copy(conn, conn)
				atomic.AddInt32(&openConns, -1)
			}
		}
	}()

	b := []byte("Hi There")

	block, _ := kcp.NewNoneBlockCrypt(nil)
	dial := cmux.Dialer(&cmux.DialerOpts{
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

	conn, err := dial("tcp", l.Addr().String())
	if !assert.NoError(t, err, "Unable to dial good conn") {
		return
	}
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

	c2, err := dial("tcp", l.Addr().String())
	if !assert.NoError(t, err) {
		return
	}
	// Immediately close c2 before writing to test that idletiming on server works
	// okay.
	c2.Close()

	time.Sleep(idleTimeout * 4)
	assert.EqualValues(t, 0, atomic.LoadInt32(&openConns), "All connections should have been closed")
}
