package water

import (
	"bytes"
	"context"
	"embed"
	"encoding/base64"
	"io"
	"log/slog"
	"net"
	"testing"

	"github.com/refraction-networking/water"
	_ "github.com/refraction-networking/water/transport/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed testdata/*
var testData embed.FS

func TestWATERListener(t *testing.T) {
	f, err := testData.Open("testdata/reverse_v1_tinygo.wasm")
	require.Nil(t, err)

	wasm, err := io.ReadAll(f)
	require.Nil(t, err)

	b64WASM := base64.StdEncoding.EncodeToString(wasm)

	ctx := context.Background()
	transport := "reverse_v1_tinygo"

	cfg := &water.Config{
		TransportModuleBin: wasm,
		OverrideLogger:     slog.New(newLogHandler(log, transport)),
	}

	l0, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	require.Nil(t, err, "ListenTCP failed: %v", err)

	ll, err := NewWATERListener(ctx, transport, l0, b64WASM)
	require.Nil(t, err)

	messageRequest := "hello"
	expectedResponse := "world"
	// running listener
	go func() {
		for {
			var conn net.Conn
			conn, err = ll.Accept()
			if err != nil {
				t.Errorf("failed to accept connection: %v ", err)
				return
			}

			go func() {
				for {
					if conn == nil {
						log.Error("nil connection")
						return
					}
					buf := make([]byte, 2*len(messageRequest))
					n, err := conn.Read(buf)
					if err != nil {
						if err == io.EOF {
							log.Debug("EOF")
							return
						}
						log.Errorf("error reading: %v", err)
						return
					}

					buf = buf[:n]
					log.Debugf("received %v", buf)
					if !bytes.Equal(buf, []byte(messageRequest)) {
						conn.Write([]byte{})
						log.Errorf("unexpected request %v %v", buf, messageRequest)
						return
					}
					n, err = conn.Write([]byte(expectedResponse))
					if err != nil {
						log.Errorf("error writing response: %v", err)
						return
					}
					log.Debugf("sent %d bytes", n)
				}
			}()
		}
	}()

	dialer, err := water.NewDialerWithContext(ctx, cfg)
	require.Nil(t, err)

	conn, err := dialer.DialContext(ctx, "tcp", l0.Addr().String())
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
