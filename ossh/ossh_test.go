package ossh

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"net"
	"os"
	"testing"

	"github.com/getlantern/ossh"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
)

func TestWrap(t *testing.T) {
	keyword := "obfuscation-keyword"
	_hostKey, err := rsa.GenerateKey(rand.Reader, 1024)
	require.NoError(t, err)
	hostKey, err := ssh.NewSignerFromKey(_hostKey)
	require.NoError(t, err)

	keyFile, err := os.CreateTemp("", "http-proxy-lantern--ossh--TestWrap")
	require.NoError(t, err)
	defer os.Remove(keyFile.Name())

	hostKeyDER := x509.MarshalPKCS1PrivateKey(_hostKey)
	require.NoError(t, pem.Encode(keyFile, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: hostKeyDER}))

	tcpL, err := net.Listen("tcp", "")
	require.NoError(t, err)
	defer tcpL.Close()

	l, err := Wrap(tcpL, keyword, keyFile.Name())
	require.NoError(t, err)
	defer l.Close()

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- func() error {
			conn, err := l.Accept()
			if err != nil {
				return fmt.Errorf("accept error: %w", nil)
			}
			defer conn.Close()

			// Echo everything sent to the server.
			_, err = io.Copy(conn, conn)
			return err
		}()
	}()

	dCfg := ossh.DialerConfig{ObfuscationKeyword: keyword, ServerPublicKey: hostKey.PublicKey()}
	conn, err := ossh.Dial("tcp", l.Addr().String(), dCfg)
	require.NoError(t, err)
	defer conn.Close()

	msg := []byte("hello ossh")
	_, err = conn.Write(msg)
	require.NoError(t, err)
	buf := make([]byte, len(msg)*2)
	n, err := conn.Read(buf)
	require.NoError(t, err)
	require.Equal(t, string(msg), string(buf[:n]))
	require.NoError(t, conn.Close())
	require.NoError(t, <-serverErr)
}
