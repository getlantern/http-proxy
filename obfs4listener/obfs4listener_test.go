package obfs4listener

import (
	"io"
	"io/ioutil"
	"net"
	"os"
	"testing"
	"time"

	"github.com/Yawning/obfs4/transports/obfs4"
	"github.com/stretchr/testify/assert"
)

func TestRoundTrip(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "obfs4listener-test")
	if !assert.NoError(t, err, "Unable to create tempdir") {
		return
	}
	defer os.RemoveAll(tmpDir)

	l, err := NewListener("localhost:0", tmpDir)
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

	tr := &obfs4.Transport{}
	cf, err := tr.ClientFactory("")
	if !assert.NoError(t, err, "Unable to create client factory") {
		return
	}

	args, err := cf.ParseArgs(l.(*obfs4listener).sf.Args())
	if !assert.NoError(t, err, "Unable to parse client args") {
		return
	}

	conn, err := cf.Dial("tcp", l.Addr().String(), net.Dial, args)
	if !assert.NoError(t, err, "Unable to dial out") {
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

func TestTimeout(t *testing.T) {
	origHandshakeTimeout := handshakeTimeout
	defer func() {
		handshakeTimeout = origHandshakeTimeout
	}()
	handshakeTimeout = 25 * time.Millisecond

	tmpDir, err := ioutil.TempDir("", "obfs4listener-test-timeout")
	if !assert.NoError(t, err, "Unable to create tempdir") {
		return
	}
	defer os.RemoveAll(tmpDir)

	l, err := NewListener("localhost:0", tmpDir)
	if !assert.NoError(t, err, "Unable to create listener") {
		return
	}
	defer l.Close()

	// Use just a regular dialer, since we have no intention of handshaking
	conn, err := net.Dial("tcp", l.Addr().String())
	if !assert.NoError(t, err, "Unable to dial") {
		return
	}
	defer conn.Close()

	start := time.Now()
	conn2, err2 := l.Accept()
	delta := time.Now().Sub(start)
	assert.Nil(t, conn2, "Failed connection should be nil")
	assert.Error(t, err2, "Failed connection should have error")
	assert.True(t, delta < 50*time.Millisecond, "Failure should have happened quickly")
}
