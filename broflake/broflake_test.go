package broflake

import (
	"net"
	"testing"

	_ "embed"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed testdata/cert.pem
var testCertPEM string

//go:embed testdata/key.pem
var testKeyPEM string

func TestWrapBroflake(t *testing.T) {
	messageRequest := "hello"
	messageResponse := "world"

	var tests = []struct {
		name               string
		givenListener      net.Listener
		givenCertPEM       string
		givenKeyPEM        string
		givenAddr          string
		assertRequestConn  func(t *testing.T, conn net.Conn)
		assertResponseConn func(t *testing.T, conn net.Conn)
	}{
		{
			name:         "wrap listener with certs",
			givenCertPEM: testCertPEM,
			givenKeyPEM:  testKeyPEM,
			givenAddr:    "127.0.0.1:8080",
			assertRequestConn: func(t *testing.T, conn net.Conn) {
				require.NotNil(t, conn)

				buf := make([]byte, 2*len(messageRequest))
				n, err := conn.Read(buf)
				assert.NoError(t, err)

				buf = buf[:n]
				assert.Equal(t, buf, []byte(messageRequest))
				conn.Write([]byte(messageResponse))
			},
			assertResponseConn: func(t *testing.T, conn net.Conn) {
				require.NotNil(t, conn)

				buf := make([]byte, 2*len(messageResponse))
				n, err := conn.Read(buf)
				assert.NoError(t, err)
				assert.Greater(t, n, 0)
			},
		},
		{
			name:      "wrap listener without certs",
			givenAddr: "127.0.0.1:8081",
			assertRequestConn: func(t *testing.T, conn net.Conn) {
				require.NotNil(t, conn)

				buf := make([]byte, 2*len(messageRequest))
				n, err := conn.Read(buf)
				assert.NoError(t, err)

				buf = buf[:n]
				assert.Equal(t, buf, []byte(messageRequest))
				conn.Write([]byte(messageResponse))
			},
			assertResponseConn: func(t *testing.T, conn net.Conn) {
				require.NotNil(t, conn)

				buf := make([]byte, 2*len(messageResponse))
				n, err := conn.Read(buf)
				assert.NoError(t, err)
				assert.Greater(t, n, 0)

				// buf = buf[:n]
				// assert.Equal(t, buf, []byte(messageResponse))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l, err := net.Listen("tcp", tt.givenAddr)
			require.NoError(t, err)

			broflakeListener, err := Wrap(l, tt.givenCertPEM, tt.givenKeyPEM)
			require.NoError(t, err)
			defer broflakeListener.Close()

			// running listener
			go func() {
				for {
					var conn net.Conn
					conn, err = broflakeListener.Accept()
					require.NoError(t, err)

					go func() {
						tt.assertRequestConn(t, conn)
					}()
				}
			}()

			conn, err := net.Dial("tcp", broflakeListener.Addr().String())
			require.NoError(t, err)
			require.NotNil(t, conn)

			n, err := conn.Write([]byte(messageRequest))
			assert.NoError(t, err)
			assert.Equal(t, len(messageRequest), n)

			tt.assertResponseConn(t, conn)
		})
	}
}
