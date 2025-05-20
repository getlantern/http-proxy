package tlsmasq

import (
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"io"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/getlantern/keyman"
	"github.com/getlantern/tlsmasq"
	"github.com/getlantern/tlsmasq/ptlshs"
)

func TestWrap(t *testing.T) {
	listenerGroup := new(sync.WaitGroup)
	defer listenerGroup.Wait()

	var secretBytes [52]byte
	rand.Read(secretBytes[:])

	secretString := hex.EncodeToString(secretBytes[:])

	proxyPrivateKey, err := keyman.GeneratePK(2048)
	require.NoError(t, err)

	proxyCert, err := proxyPrivateKey.TLSCertificateFor(time.Now().Add(10*time.Hour), false, nil, "org", "name")
	require.NoError(t, err)

	proxyCertFile := "proxy-cert.pem"
	proxyKeyFile := "proxy-key.pem"
	err = proxyCert.WriteToFile(proxyCertFile)
	require.NoError(t, err)

	err = proxyPrivateKey.WriteToFile(proxyKeyFile)
	require.NoError(t, err)

	originPrivateKey, err := keyman.GeneratePK(2048)
	require.NoError(t, err)

	originCertKeyman, err := originPrivateKey.TLSCertificateFor(time.Now().Add(10*time.Hour), false, nil, "org", "name")
	require.NoError(t, err)

	originCert, err := tls.X509KeyPair(originCertKeyman.PEMEncoded(), originPrivateKey.PEMEncoded())
	require.NoError(t, err)

	// all curve Ids, except for X25519MLKEM768, which breaks some of these TLSMasq tests for some reason
	curvePreferences := []tls.CurveID{tls.CurveP256, tls.CurveP384, tls.CurveP521, tls.X25519}

	proxiedListener, err := tls.Listen("tcp", "localhost:0",
		&tls.Config{
			Certificates:     []tls.Certificate{originCert},
			CurvePreferences: curvePreferences,
		},
	)
	require.NoError(t, err)
	defer proxiedListener.Close()

	listenerGroup.Add(1)
	go func() {
		defer listenerGroup.Done()
		for {
			conn, err := proxiedListener.Accept()
			if err != nil {
				if !strings.Contains(err.Error(), "use of closed network connection") {
					t.Log("accept error:", err)
				}
				return
			}
			defer conn.Close()
			assert.NoError(t, conn.(*tls.Conn).Handshake())
		}
	}()

	l, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)

	nonFatalErrorsHandler := func(err error) {
		assert.NoError(t, err, "got error from nonFatalErrorsHandler")
	}

	tlsmasqListener, err := Wrap(
		l, proxyCertFile, proxyKeyFile, proxiedListener.Addr().String(), secretString,
		tls.VersionTLS12, []uint16{tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA}, nonFatalErrorsHandler)
	require.NoError(t, err)
	defer tlsmasqListener.Close()

	listenerGroup.Add(1)
	go func() {
		defer listenerGroup.Done()
		for {
			conn, err := tlsmasqListener.Accept()
			if err != nil {
				if !strings.Contains(err.Error(), "use of closed network connection") {
					t.Log("accept error:", err)
				}
				return
			}
			go io.Copy(conn, conn)
		}
	}()

	insecureTLSConfig := &tls.Config{
		InsecureSkipVerify: true,
		Certificates:       []tls.Certificate{originCert},
		CurvePreferences:   curvePreferences,
	}
	dialerCfg := tlsmasq.DialerConfig{
		ProxiedHandshakeConfig: ptlshs.DialerConfig{
			Handshaker: ptlshs.StdLibHandshaker{
				Config: insecureTLSConfig,
			},
			Secret: secretBytes,
		},
		TLSConfig: insecureTLSConfig,
	}

	timeout := time.Second
	maxDataLen := 100

	dialerGroup := new(sync.WaitGroup)
	dialerGroup.Add(50)
	for i := 0; i < 50; i++ {
		go func() {
			defer dialerGroup.Done()

			conn, err := tlsmasq.DialTimeout("tcp", l.Addr().String(), dialerCfg, timeout)
			require.NoError(t, err)
			conn.SetDeadline(time.Now().Add(timeout))
			defer conn.Close()

			b := make([]byte, maxDataLen)
			rand.Read(b)

			_, err = conn.Write(b[:1])
			if !assert.NoError(t, err) {
				return
			}

			b2 := make([]byte, 1)
			_, err = io.ReadFull(conn, b2)
			if !assert.NoError(t, err) {
				return
			}
			assert.EqualValues(t, b[:1], b2)

			_, err = conn.Write(b)
			if !assert.NoError(t, err) {
				return
			}

			b3 := make([]byte, len(b))
			_, err = io.ReadFull(conn, b3)
			if !assert.NoError(t, err) {
				return
			}
			assert.EqualValues(t, b, b3)
		}()
	}
	dialerGroup.Wait()
}
