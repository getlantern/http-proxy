package obfs4listener

import (
	"io"
	"io/ioutil"
	"net"
	_ "net/http/pprof"
	"os"
	"testing"

	"git.torproject.org/pluggable-transports/obfs4.git/transports/obfs4"
	"github.com/getlantern/connmux"
	"github.com/stretchr/testify/assert"
)

func TestRoundTrip(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "obfs4listener-test")
	if !assert.NoError(t, err, "Unable to create tempdir") {
		return
	}
	defer os.RemoveAll(tmpDir)

	wrapped, err := net.Listen("tcp", "localhost:0")
	if !assert.NoError(t, err, "Unable to create listener") {
		return
	}

	ol, err := Wrap(wrapped, tmpDir)
	if !assert.NoError(t, err, "Unable to wrap listener") {
		return
	}

	pool := connmux.NewBufferPool(100)
	l := connmux.WrapListener(ol, 2, pool)
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

	args, err := cf.ParseArgs(ol.(*obfs4listener).sf.Args())
	if !assert.NoError(t, err, "Unable to parse client args") {
		return
	}

	dial := connmux.Dialer(2, pool, func() (net.Conn, error) {
		return cf.Dial("tcp", l.Addr().String(), net.Dial, args)
	})

	// Dial with a regular dialer that won't handshake. This makes sure that we're
	// handling this case correctly and not interfering with successful
	// connections afterwards
	badConn, err := net.Dial("tcp", l.Addr().String())
	if !assert.NoError(t, err, "Unable to dial bad conn") {
		return
	}
	defer badConn.Close()

	conn, err := dial()
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
