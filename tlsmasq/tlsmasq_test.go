package tlsmasq

import (
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/getlantern/keyman"
	"github.com/getlantern/tlsmasq"
	"github.com/getlantern/tlsmasq/ptlshs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWrap(t *testing.T) {
	var secretBytes [52]byte
	rand.Read(secretBytes[:])

	secretString := hex.EncodeToString(secretBytes[:])

	proxyPrivateKey, err := keyman.GeneratePK(2048)
	if !assert.NoError(t, err) {
		return
	}

	proxyCert, err := proxyPrivateKey.TLSCertificateFor(time.Now().Add(10*time.Hour), false, nil, "org", "name")
	if !assert.NoError(t, err) {
		return
	}

	proxyCertFile := "proxy-cert.pem"
	proxyKeyFile := "proxy-key.pem"
	err = proxyCert.WriteToFile(proxyCertFile)
	if !assert.NoError(t, err) {
		return
	}

	err = proxyPrivateKey.WriteToFile(proxyKeyFile)
	if !assert.NoError(t, err) {
		return
	}

	originPrivateKey, err := keyman.GeneratePK(2048)
	if !assert.NoError(t, err) {
		return
	}

	originCertKeyman, err := originPrivateKey.TLSCertificateFor(time.Now().Add(10*time.Hour), false, nil, "org", "name")
	if !assert.NoError(t, err) {
		return
	}

	originCert, err := tls.X509KeyPair(originCertKeyman.PEMEncoded(), originPrivateKey.PEMEncoded())

	wg := new(sync.WaitGroup)
	proxiedListener, err := tls.Listen("tcp", "localhost:0", &tls.Config{Certificates: []tls.Certificate{originCert}})
	defer proxiedListener.Close()

	go func() {
		for {
			conn, err := proxiedListener.Accept()
			defer conn.Close()
			require.NoError(t, err)
			require.NoError(t, conn.(*tls.Conn).Handshake())
		}
	}()

	l, err := net.Listen("tcp", "localhost:0")
	if !assert.NoError(t, err) {
		return
	}

	nonFatalErrorsHandler := func(err error) {
		assert.NoError(t, err)
	}

	tlsmasqListener, err := Wrap(l, proxyCertFile, proxyKeyFile, proxiedListener.Addr().String(), secretString, nonFatalErrorsHandler)
	if !assert.NoError(t, err) {
		return
	}
	defer tlsmasqListener.Close()

	go func() {
		for {
			conn, acceptErr := tlsmasqListener.Accept()
			if acceptErr != nil {
				return
			}
			go io.Copy(conn, conn)
		}
	}()

	insecureTLSConfig := &tls.Config{InsecureSkipVerify: true, Certificates: []tls.Certificate{originCert}}
	dialerCfg := tlsmasq.DialerConfig{
		ProxiedHandshakeConfig: ptlshs.DialerConfig{
			TLSConfig: insecureTLSConfig,
			Secret:    secretBytes,
		},
		TLSConfig: insecureTLSConfig,
	}

	timeout := time.Second
	maxDataLen := 100

	wg.Add(100)
	for i := 0; i < 100; i++ {
		go func() {
			defer wg.Done()
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

	wg.Wait()
}
