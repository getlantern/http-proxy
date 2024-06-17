package water

import (
	"bytes"
	"context"
	"net"
	"os"
	"testing"

	"github.com/refraction-networking/water"
	_ "github.com/refraction-networking/water/transport/v0"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReverseListener(t *testing.T) {
	addr := "127.0.0.1:8888"

	wasmPath := "../wasm/reverse.go.wasm"
	wasm, err := os.ReadFile(wasmPath)
	require.Nil(t, err)

	ctx := context.Background()

	cfg := &water.Config{
		TransportModuleBin: wasm,
	}

	ll, err := NewReverseListener(ctx, addr, wasm)
	require.Nil(t, err)

	messageRequest := "hello"
	expectedResponse := "world"
	// running listener
	go func() {
		for {
			var conn net.Conn
			conn, err = ll.Accept()
			if err != nil {
				t.Error(err)
			}

			go func() {
				defer conn.Close()
				buf := make([]byte, 2*len(messageRequest))
				n, err := conn.Read(buf)
				if err != nil {
					log.Errorf("error reading: %v", err)
					return
				}

				buf = buf[:n]
				if !bytes.Equal(buf, []byte(messageRequest)) {
					log.Errorf("unexpected request %v %v", buf, messageRequest)
					return
				}
				conn.Write([]byte(expectedResponse))
			}()
		}
	}()

	dialer, err := water.NewDialerWithContext(ctx, cfg)
	require.Nil(t, err)

	conn, err := dialer.DialContext(ctx, "tcp", ll.Addr().String())
	require.Nil(t, err)
	defer conn.Close()

	n, err := conn.Write([]byte(messageRequest))
	assert.Nil(t, err)
	assert.Equal(t, len(messageRequest), n)

	buf := make([]byte, 1024)
	n, err = conn.Read(buf)
	assert.Nil(t, err)
	assert.Equal(t, len(expectedResponse), n)
	assert.Equal(t, expectedResponse, string(buf[:n]))
}
