package shadowsocks

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"net"
	"testing"
	"time"

	logging "github.com/op/go-logging"

	"github.com/getlantern/fdcount"
	"github.com/getlantern/grtrack"

	"github.com/Jigsaw-Code/outline-sdk/transport"
	"github.com/Jigsaw-Code/outline-sdk/transport/shadowsocks"
	"github.com/Jigsaw-Code/outline-ss-server/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	logging.SetLevel(logging.INFO, "")
}

func makeTestCiphers(secrets []string) (service.CipherList, error) {
	configs := make([]CipherConfig, len(secrets))
	for i, secret := range secrets {
		configs[i].Secret = secret
	}

	cipherList, err := NewCipherListWithConfigs(configs)
	return cipherList, err
}

// makeTestSecrets returns a slice of `n` test passwords.  Not secure!
func makeTestSecrets(n int) []string {
	secrets := make([]string, n)
	for i := 0; i < n; i++ {
		secrets[i] = fmt.Sprintf("secret-%v", i)
	}
	return secrets
}

// tests interception of upstream connection
func TestLocalUpstreamHandling(t *testing.T) {
	req := make([]byte, 1024)
	res := make([]byte, 2048)

	_, err := rand.Read(req)
	require.Nil(t, err, "Failed to generate random request")
	_, err = rand.Read(res)
	require.Nil(t, err, "Failed to generate random response")

	l0, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	require.Nil(t, err, "ListenTCP failed: %v", err)
	secrets := makeTestSecrets(1)
	cipherList, err := makeTestCiphers(secrets)
	require.Nil(t, err, "MakeTestCiphers failed: %v", err)

	replayCache := service.NewReplayCache(1)
	options := &ListenerOptions{
		Listener:           &tcpListenerAdapter{l0},
		Ciphers:            cipherList,
		Timeout:            200 * time.Millisecond,
		ReplayCache:        &replayCache,
		ShadowsocksMetrics: &service.NoOpTCPMetrics{},
	}

	ciphers := cipherList.SnapshotForClientIP(net.ParseIP("127.0.0.1"))
	require.NotEmpty(t, ciphers, "No ciphers available")
	require.NotEmpty(t, ciphers[0].Value.(*service.CipherEntry).CryptoKey, "No crypto key available")

	accept := func(c transport.StreamConn) error {
		buf := make([]byte, 2*len(req))
		_, conn, connErr := service.NewShadowsocksStreamAuthenticator(cipherList, &replayCache, &service.NoOpTCPMetrics{})(c)
		if connErr != nil {
			log.Errorf("failed to create authenticated connection: %v", connErr)
			return connErr
		}
		defer conn.Close()
		n, err := conn.Read(buf)
		if err != nil {
			log.Errorf("error reading: %v", err)
			return err
		}
		request := buf[7:n]
		if !bytes.Equal(request, req) {
			log.Errorf("unexpected request %v %v, len buf: %d, len req: %d", request, req, len(request), len(req))
			return err
		}
		conn.Write(res)
		return nil
	}
	options.Accept = accept

	l1 := ListenLocalTCPOptions(options)
	defer l1.Close()

	client, err := shadowsocks.NewStreamDialer(
		&transport.TCPEndpoint{Address: l1.Addr().String()},
		ciphers[0].Value.(*service.CipherEntry).CryptoKey,
	)
	require.Nil(t, err, "error creating client")

	conn, err := client.DialStream(context.Background(), "127.0.0.1:443")
	require.Nil(t, err, "failed to dial")
	defer conn.Close()

	n, err := conn.Write(req)
	require.Nil(t, err, "failed to write request")
	assert.Greater(t, n, 0, "failed to write request")

	buf := make([]byte, 2*len(res))
	n, err = conn.Read(buf)
	require.Nil(t, err, "failed to read response")
	require.Equal(t, res, buf[:n], "unexpected response")
}

func TestConcurrentLocalUpstreamHandling(t *testing.T) {
	grtracker := grtrack.Start()
	_, fdc, err := fdcount.Matching("TCP")
	if err != nil {
		t.Fatal(err)
	}

	clients := 50
	reqLen := 64
	resLen := 512

	// create a request-response pair for each client
	reqs := make([][]byte, clients)
	ress := make(map[string]string)
	for i := 0; i < clients; i++ {
		req := make([]byte, reqLen)
		_, err := rand.Read(req)
		require.Nil(t, err, "Failed to generate random request")

		res := make([]byte, resLen)
		_, err = rand.Read(res)
		require.Nil(t, err, "Failed to generate random response")

		reqs[i] = req
		ress[string(req)] = string(res)
	}

	l0, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	require.Nil(t, err, "ListenTCP failed: %v", err)
	secrets := makeTestSecrets(1)
	cipherList, err := makeTestCiphers(secrets)
	require.Nil(t, err, "MakeTestCiphers failed: %v", err)

	replayCache := service.NewReplayCache(1)
	options := &ListenerOptions{
		Listener:           &tcpListenerAdapter{l0},
		Ciphers:            cipherList,
		Timeout:            200 * time.Millisecond,
		ReplayCache:        &replayCache,
		ShadowsocksMetrics: &service.NoOpTCPMetrics{},
	}
	accept := func(c transport.StreamConn) error {
		defer c.Close()
		_, conn, connErr := service.NewShadowsocksStreamAuthenticator(cipherList, &replayCache, &service.NoOpTCPMetrics{})(c)
		if connErr != nil {
			log.Errorf("failed to create authenticated connection: %v", connErr)
			return connErr
		}
		buf := make([]byte, 2*reqLen)
		n, err := conn.Read(buf)
		if err != nil {
			log.Errorf("error reading: %v", err)
			return err
		}
		request := buf[7:n]

		res := ress[string(request)]
		if res == "" {
			log.Errorf("unexpected request %v", request)
			return err
		}
		conn.Write([]byte(res))
		return nil
	}
	options.Accept = accept

	l1 := ListenLocalTCPOptions(options)

	tryReq := func(rnum int) error {
		req := reqs[rnum]
		res := []byte(ress[string(req)])

		ciphers := cipherList.SnapshotForClientIP(net.ParseIP("127.0.0.1"))
		require.NotEmpty(t, ciphers, "No ciphers available")
		require.NotEmpty(t, ciphers[0].Value.(*service.CipherEntry).CryptoKey, "No crypto key available")
		client, err := shadowsocks.NewStreamDialer(&transport.TCPEndpoint{Address: l1.Addr().String()}, ciphers[0].Value.(*service.CipherEntry).CryptoKey)
		if err != nil {
			return err
		}

		conn, err := client.DialStream(context.Background(), "127.0.0.1:443")
		if err != nil {
			return err
		}
		defer conn.Close()

		_, err = conn.Write(req)
		if err != nil {
			return err
		}

		buf := make([]byte, 2*resLen)
		n, err := conn.Read(buf)
		if err != nil {
			return err
		}
		if !bytes.Equal(res, buf[:n]) {
			return fmt.Errorf("unexpected response for req %d", rnum)
		}

		return nil
	}

	errors := make(chan error, clients)
	for i := 0; i < clients; i++ {
		id := i
		go func() {
			errors <- tryReq(id)
		}()
	}

	for i := 0; i < clients; i++ {
		require.Nil(t, <-errors, "Failed request")
	}

	l1.Close()
	require.Nil(t, fdc.AssertDelta(0), "After closing listener, there should be no lingering file descriptors")
	grtracker.Check(t)
}
