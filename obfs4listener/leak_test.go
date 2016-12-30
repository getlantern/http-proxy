package obfs4listener

import (
	"io"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"git.torproject.org/pluggable-transports/obfs4.git/transports/obfs4"
	"github.com/getlantern/cmux"
	"github.com/getlantern/http-proxy-lantern/kcplistener"
	"github.com/getlantern/idletiming"
	"github.com/getlantern/snappyconn"
	"github.com/xtaci/kcp-go"
)

// Capitalize the T to run theses test. Look at the heap and blocking
// profiles at end of test to see if there's any memory leaking.
func testTCPOBFS4ManyClients(t *testing.T) {
	doTestMany(t, true, func() (net.Listener, error) {
		return net.Listen("tcp", "localhost:0")
	}, net.Dial)
}

func testKCPOBFS4ManyClients(t *testing.T) {
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

	doTestMany(t, true, func() (net.Listener, error) {
		return kcplistener.NewListener("localhost:0")
	}, func(network, addr string) (net.Conn, error) {
		return dial(network, addr)
	})
}

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

	doTestMany(t, false, func() (net.Listener, error) {
		return kcplistener.NewListener("localhost:0")
	}, func(network, addr string) (net.Conn, error) {
		return dial(network, addr)
	})
}

func doTestMany(t *testing.T, useOBFS4 bool, listen func() (net.Listener, error), doDial func(network, addr string) (net.Conn, error)) {
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

	l, err := listen()
	if err != nil {
		t.Fatal(err)
	}

	l = idletiming.Listener(l, 15*time.Second, func(conn net.Conn) {
		conn.Close()
	})

	if useOBFS4 {
		l, err = Wrap(l, tmpDir)
		if err != nil {
			t.Fatal(err)
		}
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

	dial := doDial
	if useOBFS4 {
		tr := &obfs4.Transport{}
		cf, err := tr.ClientFactory("")
		if err != nil {
			t.Fatal(err)
		}

		args, err := cf.ParseArgs(l.(*obfs4listener).sf.Args())
		if err != nil {
			t.Fatal(err)
		}

		dial = func(network, addr string) (net.Conn, error) {
			return cf.Dial(network, addr, doDial, args)
		}
	}

	var danglingConns []net.Conn
	numClients := maxPendingHandshakesPerClient
	var wg sync.WaitGroup
	wg.Add(numClients)
	requests := make(chan bool, numClients)
	pendingDials := int64(0)
	pendingWrites := int64(0)
	for i := 0; i < numClients; i++ {
		go func() {
			defer wg.Done()

			for range requests {
				atomic.AddInt64(&pendingDials, 1)
				conn, err := dial("tcp", l.Addr().String())
				if err != nil {
					t.Fatal(err)
				}
				atomic.AddInt64(&pendingDials, -1)

				atomic.AddInt64(&pendingWrites, 1)
				_, err = conn.Write(data)
				if err != nil {
					t.Fatal(err)
				}
				atomic.AddInt64(&pendingWrites, -1)

				e := make([]byte, len(data))
				_, err = conn.Read(e)
				if err != nil && err != io.EOF {
					t.Fatal(err)
				}
				if rand.Float64() > 0.5 {
					// Half of the time, we leave the connection dangling
					danglingConns = append(danglingConns, conn)
					continue
				}
				conn.Close()
			}
		}()
	}

	go func() {
		for {
			time.Sleep(5 * time.Second)
			log.Debugf("Pending dials: %d   Pending writes: %d", atomic.LoadInt64(&pendingDials), atomic.LoadInt64(&pendingWrites))
		}
	}()

	// Note - setting the multiplier to 10 works, but at 100, the dials for
	// OBFS4/KCP start failing. I think that's because we're not reading from the
	// dangling streams and eventually the underlying session/conn gets saturated.
	for i := 0; i < numClients*100; i++ {
		requests <- true
	}
	close(requests)
	log.Debug("Done sending requests")

	wg.Wait()
	log.Debug("Done waiting")

	l.Close()
	log.Debug("Closed listener")
	go func() {
		for {
			runtime.GC()
			time.Sleep(5 * time.Second)
		}
	}()

	time.Sleep(1 * time.Minute)
	for _, conn := range danglingConns {
		conn.Close()
	}
	log.Debug("Closed dangling conns")

	time.Sleep(5 * time.Minute)
}
