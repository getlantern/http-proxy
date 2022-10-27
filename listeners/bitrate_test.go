package listeners

import (
	"fmt"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const (
	bitrateLimit = 100
)

func handleConn(t *testing.T, c net.Conn, bytesReadChan *chan int) {
	for {
		b := make([]byte, 512)
		n, err := c.Read(b)
		if err != nil {
			t.Fatal("Error reading from local connection")
		}

		if n != 0 {
			*bytesReadChan <- n
		} else {
			break
		}
	}
}

func server(t *testing.T, ready *chan struct{}, bytesReadChan *chan int) *bitrateConn {
	ln, err := net.Listen("tcp", ":9999")
	if err != nil {
		t.Fatal("Error creating listener")
	}
	bl := NewBitrateListener(ln)
	assert.NotNil(t, bl, "Should be created successfully")

	*ready <- struct{}{}

	conn, err := bl.Accept()
	conn.(*bitrateConn).ControlMessage("throttle", NewRateLimiter(bitrateLimit, bitrateLimit))

	go handleConn(t, conn, bytesReadChan)

	return conn.(*bitrateConn)
}

func TestLimited(t *testing.T) {
	ready := make(chan struct{})
	bytesReadChan := make(chan int)
	go server(t, &ready, &bytesReadChan)
	<-ready

	conn, err := net.Dial("tcp", "127.0.0.1:9999")
	if err != nil {
		t.Fatalf("Error connecting to local server: %v", err)
	}

	b := make([]byte, 101)
	for i := range b {
		b[i] = '#'
	}
	n, err := conn.Write(b)
	if err != nil {
		t.Fatalf("Error writing to connection: %v", err)
	}
	fmt.Printf("Written %v bytes\n", n)

	timer := time.NewTimer(950 * time.Millisecond)

	totalRead := 0
Done:
	for {
		select {
		case <-timer.C:
			break Done
		case nread := <-bytesReadChan:
			totalRead += nread
		}
	}

	// the limiters token bucket algorithm starts "full", so
	// we get the limit + the amount permitted in 950 milliseconds.
	// (thus, the amount read should be less than twice the limit)
	assert.True(t, 2*bitrateLimit > totalRead && totalRead > bitrateLimit, "Read an unexpected number of bytes (%d)! Rate limiting is not working", totalRead)
}

var onceStd, onceInThr, onceThr sync.Once
var benchBuf []byte

func benchSrv(wg *sync.WaitGroup, useThrottle, enableBitrate bool, port string) {
	wg.Add(1)
	go func() {
		benchBuf = make([]byte, 1024*1024)
		for i := range benchBuf {
			benchBuf[i] = '#'
		}

		ln, err := net.Listen("tcp", port)
		if err != nil {
			panic(err)
		}

		li := ln
		if useThrottle {
			li = NewBitrateListener(ln)
		}

		wg.Done()

		for {
			conn, err := li.Accept()
			if err != nil {
				panic(err)
			}

			if useThrottle {
				limit := int64(1024 * 1024 * 1024)
				conn.(*bitrateConn).ControlMessage("throttle", NewRateLimiter(limit, limit))
			}

			go func() {
				for {
					b := make([]byte, 512)
					n, err := conn.Read(b)
					if err != nil && err != io.EOF {
						panic(err)
					}
					if n == 0 {
						break
					}
				}
			}()
		}
	}()
}

func BenchmarkStandardReader(b *testing.B) {
	var wg sync.WaitGroup
	onceStd.Do(func() { benchSrv(&wg, false, false, ":9990") })
	wg.Wait()

	conn, err := net.Dial("tcp", "127.0.0.1:9990")
	if err != nil {
		panic(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		conn.Write(benchBuf)
	}
}

func BenchmarkInactiveThrottledReader(b *testing.B) {
	var wg sync.WaitGroup
	onceInThr.Do(func() { benchSrv(&wg, true, false, ":9991") })
	wg.Wait()

	conn, err := net.Dial("tcp", "127.0.0.1:9991")
	if err != nil {
		panic(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		conn.Write(benchBuf)
	}
}

func BenchmarkThrottledReader(b *testing.B) {
	var wg sync.WaitGroup
	onceThr.Do(func() { benchSrv(&wg, true, true, ":9992") })
	wg.Wait()

	conn, err := net.Dial("tcp", "127.0.0.1:9992")
	if err != nil {
		panic(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		conn.Write(benchBuf)
	}
}
