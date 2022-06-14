package tlslistener

import (
	"bufio"
	"crypto/rand"
	"crypto/tls"
	"net"
	"net/http"
	"testing"
	"time"

	utls "github.com/refraction-networking/utls"
	"github.com/stretchr/testify/assert"
)

func TestAbortOnHello(t *testing.T) {
	disallowLookbackForTesting = true
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
			hl, err := Wrap(l, "../test/data/server.key", "../test/data/server.crt", "../test/testtickets", true, tc.response, false)
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
			maintainSessionTicketKey(&tls.Config{}, "../test/testtickets", func(keys [][32]byte) { ucfg.SetSessionTicketKeys(keys) })
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
				assert.Equal(t, tc.expectedErr, err.Error())
			} else {
				assert.NoError(t, err)
				defer conn.Close()
				assert.Equal(t, "microsoft.com", uconn.ConnectionState().PeerCertificates[0].Subject.CommonName)
			}
		})
	}
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
