package obfs4listener

import (
	"io"
	"io/ioutil"
	"net"
	_ "net/http/pprof"
	"os"
	"testing"

	"git.torproject.org/pluggable-transports/obfs4.git/transports/obfs4"
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

	l, err := Wrap(wrapped, tmpDir, 1, 100, DefaultHandshakeTimeout)
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

	conn, err := cf.Dial("tcp", l.Addr().String(), func(network, addr string) (net.Conn, error) {
		return net.Dial(network, addr)
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
