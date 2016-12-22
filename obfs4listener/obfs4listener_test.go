package obfs4listener

import (
	"io"
	"io/ioutil"
	"net"
	_ "net/http/pprof"
	"os"
	"testing"

	"git.torproject.org/pluggable-transports/obfs4.git/transports/obfs4"
	"github.com/getlantern/cmux"
	"github.com/getlantern/http-proxy-lantern/kcplistener"
	"github.com/getlantern/snappyconn"
	"github.com/stretchr/testify/assert"
	"github.com/xtaci/kcp-go"
)

func TestRoundTrip(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "obfs4listener-test")
	if !assert.NoError(t, err, "Unable to create tempdir") {
		return
	}
	defer os.RemoveAll(tmpDir)

	wrapped, err := kcplistener.NewListener("localhost:0")
	if !assert.NoError(t, err, "Unable to create listener") {
		return
	}

	l, err := Wrap(wrapped, tmpDir)
	if !assert.NoError(t, err, "Unable to wrap listener") {
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

	tr := &obfs4.Transport{}
	cf, err := tr.ClientFactory("")
	if !assert.NoError(t, err, "Unable to create client factory") {
		return
	}

	args, err := cf.ParseArgs(l.(*obfs4listener).sf.Args())
	if !assert.NoError(t, err, "Unable to parse client args") {
		return
	}

	block, _ := kcp.NewNoneBlockCrypt(nil)
	dial := cmux.Dialer(&cmux.DialerOpts{
		Dial: func(network, addr string) (net.Conn, error) {
			conn, err := kcp.DialWithOptions(addr, block, 10, 3)
			if err != nil {
				return nil, err
			}
			kcplistener.ApplyDefaultConnParameters(conn)
			conn.SetDSCP(0)
			conn.SetReadBuffer(4194304)
			conn.SetWriteBuffer(4194304)
			return snappyconn.Wrap(conn), nil
		},
	})

	// Dial with a regular dialer that won't handshake. This makes sure that we're
	// handling this case correctly and not interfering with successful
	// connections afterwards
	badConn, err := dial("tcp", l.Addr().String())
	if !assert.NoError(t, err, "Unable to dial bad conn") {
		return
	}
	defer badConn.Close()

	conn, err := cf.Dial("tcp", l.Addr().String(), func(network, addr string) (net.Conn, error) {
		return dial(network, addr)
	}, args)
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
}
