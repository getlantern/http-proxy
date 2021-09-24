package rdplistener

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"testing"

	utls "github.com/refraction-networking/utls"
	"github.com/stretchr/testify/assert"

	"github.com/getlantern/http-proxy-lantern/v2/instrument"
)

func TestAbortOnHello(t *testing.T) {
	disallowLookbackForTesting = true
	testCases := []struct {
		response    HandshakeReaction
		expectedErr string
	}{
		{ReflectToRDP("192.248.148.224"), ""}, // A test windows server in Vultr
		{ReflectToRDP("site.not-exist"), "EOF"},
	}

	for tn, tc := range testCases {
		t.Run(tc.response.action, func(t *testing.T) {
			l, _ := net.Listen("tcp", ":1231")
			defer l.Close()
			hl, err := Wrap(l, "../test/testtickets", true, tc.response, instrument.NoInstrument{}, "192.248.148.224")
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
						// Note to reader, this path never seems to be tested?
					}(sconn)
				}
			}()

			cfg := &tls.Config{ServerName: fmt.Sprintf("WIN-S0UMC4DONE%d", tn), InsecureSkipVerify: true}
			conn, err := net.Dial("tcp", l.Addr().String())
			_, err = conn.Write(rdpStartTLS)
			assert.NoError(t, err)
			startTLSAck := make([]byte, 1500)
			n, err := conn.Read(startTLSAck)
			assert.NoError(t, err)
			assert.Equal(t, 0, bytes.Compare(startTLSAck[:n], rdpStartTLSAck))

			// conn, err := tls.Dial("tcp", l.Addr().String(), cfg)
			tconn := tls.Client(conn, cfg)
			err = tconn.Handshake()
			// log.Print(tc.expectedErr)
			// time.Sleep(time.Hour)
			if tc.expectedErr == "" {
				assert.NoError(t, err)
			}

			if tc.expectedErr != "" {
				assert.Error(t, err)
				assert.Equal(t, tc.expectedErr, err.Error())
			} else {
				assert.NoError(t, err)
				defer conn.Close()
				// RDPHello := make([]byte, 1500)
				// n, err = tconn.Read(RDPHello)
				// log.Printf("%x", RDPHello[:n])
				// TODO: SENT NTLM Auth TO CONFIRM CORRECT REFLECTION
			}

			////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
			// time.Sleep(time.Hour)
			////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

			// Now make sure we can't spoof a session ticket.
			rawConn, err := net.Dial("tcp", l.Addr().String())
			_, err = rawConn.Write(rdpStartTLS)
			assert.NoError(t, err)
			startTLSAck = make([]byte, 1500)
			n, err = rawConn.Read(startTLSAck)
			assert.NoError(t, err)
			assert.Equal(t, 0, bytes.Compare(startTLSAck[:n], rdpStartTLSAck))

			ucfg := &utls.Config{ServerName: fmt.Sprintf("WIN-S0UMC4DTWO%d", tn), InsecureSkipVerify: true}
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
				// if err.Error() != "EOF" {
				// 	assert.NoError(t, err)
				// }
				// assert.NotEqual(t, "", uconn.ConnectionState().PeerCertificates[0].Subject.CommonName)
				defer conn.Close()
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
