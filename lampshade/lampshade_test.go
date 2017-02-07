package lampshade

import (
	"crypto/rsa"
	"io"
	"math/rand"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/getlantern/http-proxy/buffers"
	"github.com/getlantern/keyman"
	"github.com/getlantern/lampshade"
	"github.com/stretchr/testify/assert"
)

func TestRoundTrip(t *testing.T) {
	pk, err := keyman.GeneratePK(2048)
	if !assert.NoError(t, err) {
		return
	}

	cert, err := pk.TLSCertificateFor("org", "name", time.Now().Add(10*time.Hour), false, nil)
	if !assert.NoError(t, err) {
		return
	}

	certFile := "cert.pem"
	keyFile := "key.pem"
	err = cert.WriteToFile(certFile)
	if !assert.NoError(t, err) {
		return
	}

	err = pk.WriteToFile(keyFile)
	if !assert.NoError(t, err) {
		return
	}

	l, err := net.Listen("tcp", "localhost:0")
	if !assert.NoError(t, err) {
		return
	}

	l, err = Wrap(l, certFile, keyFile)
	if !assert.NoError(t, err) {
		return
	}
	defer l.Close()

	go func() {
		for {
			conn, acceptErr := l.Accept()
			if acceptErr != nil {
				return
			}
			go io.Copy(conn, conn)
		}
	}()

	certRT, err := keyman.LoadCertificateFromPEMBytes(cert.PEMEncoded())
	if !assert.NoError(t, err) {
		return
	}
	dial := lampshade.Dialer(50, 0, buffers.Pool(), certRT.X509().PublicKey.(*rsa.PublicKey), func() (net.Conn, error) {
		return net.Dial("tcp", l.Addr().String())
	})

	var wg sync.WaitGroup
	wg.Add(100)
	for i := 0; i < 100; i++ {
		go func() {
			defer wg.Done()
			conn, err := dial()
			if !assert.NoError(t, err) {
				return
			}
			defer conn.Close()

			b := make([]byte, lampshade.MaxDataLen)
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
