package starbridge

import (
	"fmt"
	"net"
	"testing"

	replicant "github.com/OperatorFoundation/Replicant-go/Replicant/v3"
	"github.com/OperatorFoundation/Replicant-go/Replicant/v3/polish"
	"github.com/OperatorFoundation/Replicant-go/Replicant/v3/toneburst"
	"github.com/OperatorFoundation/Starbridge-go/Starbridge/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWrap(t *testing.T) {
	var (
		clientMsg = "hello from the client"
		serverMsg = "hello from the server"
	)

	pub, priv, err := Starbridge.GenerateKeys()
	require.NoError(t, err)

	tcpListener, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)
	defer tcpListener.Close()

	l, err := Wrap(tcpListener, *priv)
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
			clientCfg := getClientConfig(*pub)

			tcpConn, err := net.Dial("tcp", l.Addr().String())
			if err != nil {
				return result{err: fmt.Errorf("dial error: %w", err)}
			}
			defer tcpConn.Close()

			conn, err := Starbridge.NewClientConnection(clientCfg, tcpConn)
			if err != nil {
				return result{err: fmt.Errorf("handshake error: %w", err)}
			}

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
}

// Adapted from https://github.com/OperatorFoundation/Starbridge-go/blob/v3.0.12/Starbridge/v3/starbridge.go#L237-L253
func getClientConfig(serverPublicKey string) replicant.ClientConfig {
	polishClientConfig := polish.DarkStarPolishClientConfig{
		ServerAddress:   fakeListenAddr,
		ServerPublicKey: serverPublicKey,
	}

	toneburstClientConfig := toneburst.StarburstConfig{
		Mode: "SMTPClient",
	}

	clientConfig := replicant.ClientConfig{
		Toneburst: toneburstClientConfig,
		Polish:    polishClientConfig,
	}

	return clientConfig
}
