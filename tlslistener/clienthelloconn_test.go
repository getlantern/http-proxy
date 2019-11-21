package tlslistener

import (
	"bufio"
	"crypto/rand"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	utls "github.com/getlantern/utls"
	"github.com/stretchr/testify/assert"
)

func TestAbortOnHello(t *testing.T) {
	disallowLookbackForTesting = true
	l, err := net.Listen("tcp", ":0")
	assert.NoError(t, err)

	hl, err := Wrap(l, "../test/data/server.key", "../test/data/server.crt", "../test/testtickets", true)
	assert.NoError(t, err)

	handleConnection := func(sconn net.Conn) {
		fmt.Println("Handling connection")

		buf := bufio.NewReader(sconn)
		_, err := http.ReadRequest(buf)
		if err != nil {
			return
		}
		/*
			res := http.Response{
				Status: "200 OK",
			}
			res.Write(sconn)
		*/
	}

	go func() {
		for {
			sconn, err := hl.Accept()
			//defer sconn.Close()
			assert.NoError(t, err)
			go handleConnection(sconn)

		}
	}()

	cfg := &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         "microsoft.com",
	}

	_, err = tls.Dial("tcp", l.Addr().String(), cfg)
	assert.Error(t, err)

	// Now make sure we can't spoof a session ticket.
	rawConn, err := net.DialTimeout("tcp", l.Addr().String(), 4*time.Second)

	ucfg := &utls.Config{
		ServerName: "microsoft.com",
	}
	maintainSessionTicketKey(&tls.Config{}, "../test/testtickets", func(keys [][32]byte) { ucfg.SetSessionTicketKeys(keys) })

	ss := &utls.ClientSessionState{}
	ticket := make([]byte, 120)
	rand.Read(ticket)
	ss.SetSessionTicket(ticket)
	ss.SetVers(tls.VersionTLS12)

	uconn := utls.UClient(rawConn, ucfg, utls.HelloChrome_Auto)
	uconn.SetSessionState(ss)

	req, err := http.NewRequest("get", "https://microsoft.com", nil)
	assert.NoError(t, err)
	err = req.Write(uconn)
	assert.Error(t, err)

	// Now try to get an actual response from microsoft.com.
	fmt.Printf("trying for real...")
	cfg = &tls.Config{
		ServerName: "microsoft.com",
	}
	conn, err := tls.Dial("tcp", l.Addr().String(), cfg)
	assert.NoError(t, err)
	req, _ = http.NewRequest("get", "https://microsoft.com", nil)
	err = req.Write(conn)

	assert.NoError(t, err)

	res, err := http.ReadResponse(bufio.NewReader(conn), req)
	assert.NoError(t, err)
	fmt.Printf("Response: %#v", res)

	hl.Close()
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
