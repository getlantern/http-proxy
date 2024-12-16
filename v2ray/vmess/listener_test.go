package vmess

import (
	"fmt"
	"net"
	"testing"
	"time"

	vmess "github.com/getlantern/sing-vmess"
	"github.com/sagernet/sing/common/metadata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWrap(t *testing.T) {
	securityOptions := []string{"zero", "auto", "none", "aes-128-gcm", "chacha20-poly1305"}
	for _, securityOption := range securityOptions {
		t.Run(securityOption, func(t *testing.T) {
			uuid := "3fed9a96-900c-4dd4-9fd2-f333a566768c"
			var (
				clientMsg = "hello from the client"
				serverMsg = "hello from the server"
			)

			tcpListener, err := net.Listen("tcp", "localhost:0")
			require.NoError(t, err)
			defer tcpListener.Close()

			l, err := NewVMessListener(tcpListener, []string{uuid})
			require.NoError(t, err)
			defer l.Close()

			type result struct {
				msg string
				err error
			}

			serverResC := make(chan result)
			go func() {
				serverResC <- func() result {
					conn, err := l.Accept()
					if err != nil {
						return result{err: fmt.Errorf("accept error: %w", err)}
					}
					defer conn.Close()

					buf := make([]byte, 1024)
					n, err := conn.Read(buf)
					if err != nil {
						return result{err: fmt.Errorf("read error: %w", err)}
					}

					if _, err := conn.Write([]byte(serverMsg)); err != nil {
						return result{err: fmt.Errorf("write error: %w", err)}
					}

					return result{msg: string(buf[:n])}
				}()
			}()

			clientResC := make(chan result)
			go func() {
				clientResC <- func() result {
					tcpConn, err := net.Dial("tcp", l.Addr().String())
					if err != nil {
						return result{err: fmt.Errorf("dial error: %w", err)}
					}
					defer tcpConn.Close()

					client, err := vmess.NewClient(uuid, securityOption, 0)
					require.NoError(t, err)

					target := metadata.ParseSocksaddrHostPort("random.stuff.com", 443)

					conn := client.DialEarlyConn(tcpConn, target)

					_, err = conn.Write([]byte(clientMsg))
					if err != nil {
						return result{err: fmt.Errorf("write error: %w", err)}
					}

					buf := make([]byte, 1024)
					n, err := conn.Read(buf)
					if err != nil {
						return result{err: fmt.Errorf("read error: %w", err)}
					}

					return result{msg: string(buf[:n])}
				}()
			}()

			for i := 0; i < 2; i++ {
				select {
				case res := <-clientResC:
					if assert.NoError(t, res.err) {
						assert.Equal(t, serverMsg, res.msg)
					}
				case res := <-serverResC:
					if assert.NoError(t, res.err) {
						assert.Equal(t, clientMsg, res.msg)
					}
				}
			}
		})
	}
}

func TestUUIDMismatch(t *testing.T) {
	uuid1 := "3fed9a96-900c-4dd4-9fd2-f333a5667681"
	uuid2 := "3fed9a96-900c-4dd4-9fd2-f333a5667682"

	tcpListener, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)
	defer tcpListener.Close()

	l, err := NewVMessListener(tcpListener, []string{uuid1})
	require.NoError(t, err)
	defer l.Close()

	serverResC := make(chan error)
	go func() {
		serverResC <- func() error {
			_, err := l.Accept()
			require.Error(t, err)
			return nil
		}()
	}()

	clientResC := make(chan error)
	go func() {
		clientResC <- func() error {
			tcpConn, err := net.Dial("tcp", l.Addr().String())
			if err != nil {
				return err
			}
			defer tcpConn.Close()

			client, err := vmess.NewClient(uuid2, "auto", 0)
			require.NoError(t, err)

			target := metadata.ParseSocksaddrHostPort("random.stuff.com", 443)

			conn := client.DialEarlyConn(tcpConn, target)

			_, err = conn.Write([]byte("test that will fail"))
			if err != nil {
				return fmt.Errorf("write error: %w", err)
			}

			buf := make([]byte, 1024)
			_ = conn.SetDeadline(time.Now().Add(100 * time.Millisecond))
			_, err = conn.Read(buf)
			require.Error(t, err)
			return nil
		}()
	}()

	for i := 0; i < 2; i++ {
		select {
		case res := <-clientResC:
			require.NoError(t, res)
		case res := <-serverResC:
			assert.NoError(t, res)

		}
	}
}
