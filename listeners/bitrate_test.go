package listeners

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/getlantern/testify/assert"
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
	bl := NewBitrateListener(ln, bitrateLimit)
	assert.NotNil(t, bl, "Should be created succesfully")

	*ready <- struct{}{}

	conn, err := bl.Accept()
	conn.(*bitrateConn).active = true

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

	b := make([]byte, 2048)
	for i := range b {
		b[i] = '#'
	}
	n, err := conn.Write(b)
	if err != nil {
		t.Fatalf("Error writing to connection: %v", err)
	}
	fmt.Printf("Written %v bytes\n", n)

	timer := time.NewTimer(time.Second)

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

	assert.Equal(t, bitrateLimit, totalRead, "Read an unexpected number of bytes! Rate limiting is not working")
}
