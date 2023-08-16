package tlslistener

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"io"
	mrand "math/rand"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"

	utls "github.com/refraction-networking/utls"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xtaci/smux"

	"github.com/getlantern/cmux/v2"
	"github.com/getlantern/http-proxy-lantern/v2/instrument"
)

func TestAbortOnHello(t *testing.T) {
	disallowLookbackForTesting = true
	defer func() {
		disallowLookbackForTesting = false
	}()

	testCases := []struct {
		response    HandshakeReaction
		expectedErr string
	}{
		{AlertHandshakeFailure, "remote error: tls: handshake failure"},
		{AlertProtocolVersion, "remote error: tls: protocol version not supported"},
		{AlertInternalError, "remote error: tls: internal error"},
		{CloseConnection, "EOF"},
		{ReflectToSite("microsoft.com"), ""},
		{ReflectToSite("site.not-exist"), "EOF"},

		{Delayed(100*time.Millisecond, AlertInternalError), "remote error: tls: internal error"},
		{Delayed(100*time.Millisecond, CloseConnection), "EOF"},
	}

	for _, tc := range testCases {
		t.Run(tc.response.action, func(t *testing.T) {
			l, _ := net.Listen("tcp", ":0")
			defer l.Close()
			hl, err := Wrap(
				l, "../test/data/server.key", "../test/data/server.crt", "../test/testtickets", "",
				true, tc.response, false, instrument.NoInstrument{})
			assert.NoError(t, err)
			defer hl.Close()

			go func() {
				for {
					sconn, err := hl.Accept()
					if err != nil {
						return
					}
					go func(sconn net.Conn) {
						_, err := http.ReadRequest(bufio.NewReader(sconn))
						if err != nil {
							return
						}
						(&http.Response{Status: "200 OK"}).Write(sconn)
					}(sconn)
				}
			}()

			cfg := &tls.Config{ServerName: "microsoft.com"}
			conn, err := tls.Dial("tcp", l.Addr().String(), cfg)
			if tc.expectedErr != "" {
				assert.Error(t, err)
				assert.Equal(t, tc.expectedErr, err.Error())
			} else {
				assert.NoError(t, err)
				defer conn.Close()
				assert.Equal(t, "microsoft.com", conn.ConnectionState().PeerCertificates[0].Subject.CommonName)
				req, _ := http.NewRequest("GET", "https://microsoft.com", nil)
				assert.NoError(t, req.Write(conn))
				resp, err := http.ReadResponse(bufio.NewReader(conn), req)
				assert.NoError(t, err)
				assert.Equal(t, http.StatusMovedPermanently, resp.StatusCode)
			}

			// Now make sure we can't spoof a session ticket.
			rawConn, err := net.Dial("tcp", l.Addr().String())
			assert.NoError(t, err)
			ucfg := &utls.Config{ServerName: "microsoft.com"}
			maintainSessionTicketKey(
				&tls.Config{}, "../test/testtickets", nil,
				func(keys [][32]byte) { ucfg.SetSessionTicketKeys(keys) })
			ss := &utls.ClientSessionState{}
			ticket := make([]byte, 120)
			rand.Read(ticket)
			ss.SetSessionTicket(ticket)
			ss.SetVers(tls.VersionTLS12)

			uconn := utls.UClient(rawConn, ucfg, utls.HelloChrome_Auto)
			uconn.SetSessionState(ss)
			err = uconn.Handshake()
			if tc.expectedErr != "" {
				assert.Error(t, err)
				assert.Equal(t, tc.expectedErr, err.Error(), tc.response.action)
			} else {
				assert.NoError(t, err)
				defer conn.Close()
				assert.Equal(t, "microsoft.com", uconn.ConnectionState().PeerCertificates[0].Subject.CommonName)
			}
		})
	}
}

func TestClientHelloConcurrency(t *testing.T) {
	numWorkers := 100
	requestsPerConn := 10

	l, _ := net.Listen("tcp", ":0")
	defer l.Close()
	hl, err := Wrap(
		l, "../test/data/server.key", "../test/data/server.crt", "../test/testtickets", "",
		true, AlertHandshakeFailure, false, instrument.NoInstrument{})
	assert.NoError(t, err)
	defer hl.Close()

	proto := cmux.NewSmuxProtocol(smux.DefaultConfig())
	ml := cmux.Listen(&cmux.ListenOpts{
		Listener: hl,
		Protocol: proto,
	})
	defer ml.Close()

	go func() {
		for {
			conn, err := ml.Accept()
			if err != nil {
				return
			}
			go func() {
				defer conn.Close()
				b := bufio.NewReader(conn)
				for i := 0; i < requestsPerConn; i++ {
					req, err := http.ReadRequest(b)
					if err != nil && i == 0 {
						// this is okay, this is just the first connection that gets a session ticket
						return
					}
					require.NoError(t, err)
					io.Copy(io.Discard, req.Body)
					req.Body.Close()
					var resp http.Response
					resp.StatusCode = http.StatusOK
					resp.Write(conn)
				}
			}()
		}
	}()

	var wg sync.WaitGroup
	wg.Add(numWorkers)
	for i := 0; i < numWorkers; i++ {
		go func() {
			defer wg.Done()
			cfg := &tls.Config{
				InsecureSkipVerify: true,
				ClientSessionCache: tls.NewLRUClientSessionCache(0),
			}
			// Dial once without session ticket
			var conn net.Conn
			var err error
			conn, err = tls.Dial("tcp", l.Addr().String(), cfg)
			require.NoError(t, err)
			// Write some jibberish
			jibberish := make([]byte, 1024)
			_, err = rand.Read(jibberish)
			require.NoError(t, err)
			_, err = conn.Write(jibberish)
			require.NoError(t, err)
			conn.Close()

			// Dial again to send actual requests
			md := cmux.Dialer(&cmux.DialerOpts{
				Dial: func(ctx context.Context, network, addr string) (conn net.Conn, err error) {
					return tls.Dial(network, addr, cfg)
				},
				Protocol: proto,
			})
			conn, err = md(context.Background(), "tcp", l.Addr().String())
			require.NoError(t, err)
			defer conn.Close()

			b := bufio.NewReader(conn)
			// Pipeline requests
			for i := 0; i < requestsPerConn; i++ {
				req, err := http.NewRequest("GET", "http://"+l.Addr().String(), bytes.NewReader([]byte("Hello World")))
				require.NoError(t, err)
				err = req.Write(conn)
				require.NoError(t, err)
			}
			for i := 0; i < requestsPerConn; i++ {
				req, err := http.NewRequest("GET", "http://"+l.Addr().String(), bytes.NewReader([]byte("Hello World")))
				require.NoError(t, err)
				resp, err := http.ReadResponse(b, req)
				require.NoError(t, err)
				io.Copy(io.Discard, resp.Body)
				require.Equal(t, http.StatusOK, resp.StatusCode)
			}
		}()
	}
	wg.Wait()
}

func TestParseInvalidTicket(t *testing.T) {
	scfg := &utls.Config{}
	var tk [32]byte
	rand.Read(tk[:])
	scfg.SetSessionTicketKeys([][32]byte{tk})
	ticket := make([]byte, 120)
	rand.Read(ticket)
	plainText, _ := utls.DecryptTicketWith(ticket, scfg)
	assert.Len(t, plainText, 0)
}

var maxSlowChunkSize = 50

type slowConn struct {
	net.Conn
}

func (c *slowConn) Write(b []byte) (int, error) {
	slowChunkSize := mrand.Intn(maxSlowChunkSize) + 1
	written := 0
	for i := 0; i < len(b); i += slowChunkSize {
		time.Sleep(10 * time.Millisecond)
		end := i + slowChunkSize
		if end > len(b) {
			end = len(b)
		}
		n, err := c.Conn.Write(b[i:end])
		written += n
		if err != nil {
			return written, err
		}
	}
	return written, nil
}

// func (c *slowConn) Read(b []byte) (int, error) {
// 	read := 0
// 	for i := 0; i < len(b); i += slowChunkSize {
// 		// time.Sleep(10 * time.Millisecond)
// 		end := i + slowChunkSize
// 		if end > len(b) {
// 			end = len(b)
// 		}
// 		n, err := c.Conn.Read(b[i:end])
// 		read += n
// 		if err != nil {
// 			return read, err
// 		}
// 	}
// 	fmt.Printf("%v vs %v\n", read, len(b))
// 	return read, nil
// }
