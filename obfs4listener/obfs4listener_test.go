package obfs4listener

import (
	"git.torproject.org/pluggable-transports/obfs4.git/transports/obfs4"
	"github.com/getlantern/cmux"
	"github.com/getlantern/http-proxy-lantern/kcplistener"
	"github.com/getlantern/snappyconn"
	"github.com/stretchr/testify/assert"
	"github.com/xtaci/kcp-go"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime"
	"sync"
	"testing"
	"time"
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

// Capitalize the T to run this test. Look at the heap and blocking
// profiles at end of test to see if there's any memory leaking.
func testTCPManyClients(t *testing.T) {
	doTestMany(t, func() (net.Listener, error) {
		return net.Listen("tcp", "localhost:0")
	}, net.Dial)
}

// Capitalize the T to run this test. Look at the heap and blocking
// profiles at end of test to see if there's any memory leaking.
func testKCPManyClients(t *testing.T) {
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

	doTestMany(t, func() (net.Listener, error) {
		return kcplistener.NewListener("localhost:0")
	}, func(network, addr string) (net.Conn, error) {
		return dial(network, addr)
	})
}

func doTestMany(t *testing.T, listen func() (net.Listener, error), dial func(network, addr string) (net.Conn, error)) {
	go func() {
		log.Debug("Starting pprof page at http://localhost:6060/debug/pprof")
		if err := http.ListenAndServe("localhost:6060", nil); err != nil {
			log.Error(err)
		}
	}()

	tmpDir, err := ioutil.TempDir("", "obfs4listener-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	wrapped, err := listen()
	if err != nil {
		t.Fatal(err)
	}

	l, err := Wrap(wrapped, tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	log.Debug("Opened listener")

	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				if err != io.EOF && err.Error() != "listener closed" && err.Error() != "broken pipe" || err.Error() != "use of closed network connection" {
					t.Fatal(err)
				}
				return
			}
			go func() {
				defer conn.Close()
				buf := make([]byte, 32768)
				for {
					n, err := conn.Read(buf)
					if err != nil {
						return
					}
					_, err = conn.Write(buf[:n])
					if err != nil {
						return
					}
				}
			}()
		}
	}()

	data := []byte("Hi There")

	tr := &obfs4.Transport{}
	cf, err := tr.ClientFactory("")
	if err != nil {
		t.Fatal(err)
	}

	args, err := cf.ParseArgs(l.(*obfs4listener).sf.Args())
	if err != nil {
		t.Fatal(err)
	}

	numClients := 100
	var wg sync.WaitGroup
	wg.Add(numClients)
	requests := make(chan bool, numClients)
	for i := 0; i < numClients; i++ {
		go func() {
			defer wg.Done()

			for range requests {
				conn, err := cf.Dial("tcp", l.Addr().String(), dial, args)
				if err != nil {
					t.Fatal(err)
				}

				_, err = conn.Write(data)
				if err != nil {
					t.Fatal(err)
				}
				e := make([]byte, len(data))
				_, err = conn.Read(e)
				if err != nil && err != io.EOF {
					t.Fatal(err)
				}
				conn.Close()
			}
		}()
	}

	for i := 0; i < 25000; i++ {
		requests <- true
	}
	close(requests)

	wg.Wait()
	log.Debug("Done waiting")

	l.Close()
	log.Debug("Closed listener")
	go func() {
		runtime.GC()
		time.Sleep(5 * time.Second)
	}()
	time.Sleep(5 * time.Minute)
}
