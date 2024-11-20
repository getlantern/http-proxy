package tlslistener

import (
	"bufio"
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"net"
	"net/http"
	"testing"
	"time"

	utls "github.com/refraction-networking/utls"
	"github.com/stretchr/testify/require"

	"github.com/getlantern/http-proxy-lantern/v2/instrument"
)

func TestAbortOnHello(t *testing.T) {
	allowLoopbackForTesting = true
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
				l, "../test/data/server.key", "../test/data/server.crt", "../test/testtickets", "", "",
				true, tc.response, false, instrument.NoInstrument{})
			require.NoError(t, err)
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

			cfg := &tls.Config{ServerName: "microsoft.com", InsecureSkipVerify: true}
			conn, err := tls.Dial("tcp", l.Addr().String(), cfg)
			// For now, we expect this to work always, even when we're missing a session ticket
			// See https://github.com/getlantern/engineering/issues/292#issuecomment-1765268377
			require.NoError(t, err)
			conn.Close()
			// if tc.expectedErr != "" {
			// 	require.Error(t, err)
			// 	require.Equal(t, tc.expectedErr, err.Error())
			// } else {
			// 	require.NoError(t, err)
			// 	defer conn.Close()
			// 	require.Equal(t, "microsoft.com", conn.ConnectionState().PeerCertificates[0].Subject.CommonName)
			// 	req, _ := http.NewRequest("GET", "https://microsoft.com", nil)
			// 	require.NoError(t, req.Write(conn))
			// 	resp, err := http.ReadResponse(bufio.NewReader(conn), req)
			// 	require.NoError(t, err)
			// 	require.Equal(t, http.StatusMovedPermanently, resp.StatusCode)
			// }

			// // Now make sure we can't spoof a session ticket.
			// rawConn, err := net.Dial("tcp", l.Addr().String())
			// require.NoError(t, err)
			// ucfg := &utls.Config{ServerName: "microsoft.com"}
			// maintainSessionTicketKeyFile("../test/testtickets", "",
			// 	func(keys [][32]byte) { ucfg.SetSessionTicketKeys(keys) })
			// ss := &utls.ClientSessionState{}
			// ticket := make([]byte, 120)
			// rand.Read(ticket)
			// ss.SetSessionTicket(ticket)
			// ss.SetVers(tls.VersionTLS12)

			// uconn := utls.UClient(rawConn, ucfg, utls.HelloChrome_Auto)
			// uconn.SetSessionState(ss)
			// err = uconn.Handshake()
			// if tc.expectedErr != "" {
			// 	require.Error(t, err)
			// 	require.Equal(t, tc.expectedErr, err.Error(), tc.response.action)
			// } else {
			// 	require.NoError(t, err)
			// 	defer conn.Close()
			// 	require.Equal(t, "microsoft.com", uconn.ConnectionState().PeerCertificates[0].Subject.CommonName)
			// }
		})
	}
}

func TestSuccess(t *testing.T) {
	allowLoopbackForTesting = false
	l, _ := net.Listen("tcp", ":0")
	defer l.Close()

	// We specify in-memory session ticket keys to make sure that key rotation for these is working
	// but that we are NOT requiring clients to present session tickets (yet).
	// See https://github.com/getlantern/engineering/issues/292#issuecomment-1687180508.
	sessionTicketKeys := make([]byte, keySize)
	_, err := rand.Read(sessionTicketKeys)
	require.NoError(t, err)
	strKeys := base64.StdEncoding.EncodeToString(sessionTicketKeys)

	hl, err := Wrap(
		l, "../test/data/server.key", "../test/data/server.crt", "", "", strKeys,
		true, AlertHandshakeFailure, false, instrument.NoInstrument{})
	require.NoError(t, err)
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
				(&http.Response{StatusCode: http.StatusAccepted}).Write(sconn)
			}(sconn)
		}
	}()

	// Dial once to obtain a valid session ticket (this is works because we're dialing localhost)
	ucfg := &utls.Config{
		InsecureSkipVerify: true,
		// ClientSessionCache: utls.NewLRUClientSessionCache(10),
	}
	conn, err := utls.Dial("tcp", l.Addr().String(), ucfg)
	require.NoError(t, err)
	defer conn.Close()

	// Now disallow loopback for testing, then dial again and make sure session ticket still works
	allowLoopbackForTesting = true
	defer func() {
		allowLoopbackForTesting = false
	}()

	conn, err = utls.Dial("tcp", l.Addr().String(), ucfg)
	require.NoError(t, err)
	defer conn.Close()

	req, err := http.NewRequest("GET", "/", nil)
	require.NoError(t, err)
	err = req.Write(conn)
	require.NoError(t, err)
	resp, err := http.ReadResponse(bufio.NewReader(conn), req)
	require.NoError(t, err)
	require.Equal(t, http.StatusAccepted, resp.StatusCode)
}

func TestParseInvalidTicket(t *testing.T) {
	var tk [32]byte
	rand.Read(tk[:])
	ticket := make([]byte, 120)
	rand.Read(ticket)

	utlsConfig := &utls.Config{}
	utlsConfig.SetSessionTicketKeys([][32]byte{tk})
	uss, _ := utlsConfig.DecryptTicket(ticket, utls.ConnectionState{})
	require.Nil(t, uss)
}
