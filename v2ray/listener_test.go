package v2ray

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	core "github.com/v2fly/v2ray-core/v5"
	"github.com/v2fly/v2ray-core/v5/infra/conf/serial"
	socksProxy "golang.org/x/net/proxy"
	"net"
	"net/url"
	"os"
	"strings"
	"testing"
)

func TestListener(t *testing.T) {
	tests := []struct {
		name         string
		clientConfig string
		serverConfig string
	}{
		{
			name:         "socks",
			serverConfig: "socks_server.json", clientConfig: "socks_client.json",
		},
		{
			name:         "vmess",
			serverConfig: "vmess_server.json", clientConfig: "vmess_client.json",
		}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testServer, err := os.ReadFile("fixtures/" + tt.serverConfig)
			require.NoError(t, err)
			testConfig, err := os.ReadFile("fixtures/" + tt.clientConfig)
			require.NoError(t, err)

			_, err = NewV2RayListener(context.Background(), "127.0.0.1:11082", string(testServer))

			require.Nil(t, err)

			messageRequest := "hello"
			expectedResponse := "world"

			// start tcp server that waits for messageRequest and responds with expectedResponse
			ln, err := net.Listen("tcp", "127.0.0.1:11084")
			require.NoError(t, err)
			defer ln.Close()

			go func() {
				conn, err := ln.Accept()
				if err != nil {
					t.Error(err)
					return
				}
				defer conn.Close()
				buf := make([]byte, len(messageRequest))

				if _, err = conn.Read(buf); err != nil {
					t.Error(err)
					return
				}
				if string(buf) == messageRequest {
					_, err = conn.Write([]byte(expectedResponse))
					if err != nil {
						t.Error(err)
					}
				}
			}()

			config, err := serial.LoadJSONConfig(strings.NewReader(string(testConfig)))
			require.NoError(t, err)

			server, err := core.New(config)
			require.NoError(t, err)

			err = server.Start()
			require.NoError(t, err)
			defer server.Close()

			purl, err := url.Parse(fmt.Sprintf("socks5://%s:%d", "127.0.0.1", 11083))
			require.NoError(t, err)
			socksDiler, err := socksProxy.FromURL(purl, nil)
			require.NoError(t, err)
			conn, err := socksDiler.Dial("tcp", "127.0.0.1:11084")
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
		})
	}

}
