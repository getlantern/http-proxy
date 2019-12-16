package refraction

import (
	"bufio"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/getlantern/golog"
	"github.com/getlantern/tlsdefaults"
	utls "github.com/getlantern/utls"
	"github.com/stretchr/testify/assert"
)

func TestParseHandshake(t *testing.T) {

	// Now make sure we can't spoof a session ticket.
	rawConn, err := net.DialTimeout("tcp", "microsoft.com:443", 10*time.Second)
	assert.NoError(t, err)

	ucfg := &utls.Config{
		ServerName: "microsoft.com",
	}
	uconn := utls.UClient(rawConn, ucfg, utls.HelloChrome_Auto)

	req, err := http.NewRequest("get", "https://microsoft.com", nil)
	assert.NoError(t, err)
	err = req.Write(uconn)
	fmt.Printf("%v\n\n\n\n", hex.Dump(uconn.HandshakeState.ServerHello.Raw))

	data, err := ioutil.ReadFile("../test/handshakebytes")
	assert.NoError(t, err)
	fmt.Printf("%v\n", len(data))

	fmt.Printf("%v\n", hex.Dump(data[5:127]))

	fmt.Printf("%v\n", hex.Dump(data[:127]))
	fmt.Printf("%v\n", hex.Dump(data[127:]))

	_, err = utls.UnmarshalServerHello(data[5:127])

	assert.NoError(t, err)
}

func TestAbortOnHello(t *testing.T) {

	dst := runDestinationServer()

	l, err := net.Listen("tcp", ":0")
	assert.NoError(t, err)

	hl, err := Wrap(l, dst)
	assert.NoError(t, err)

	handleConnection := func(sconn net.Conn) {
		fmt.Println("Handling connection")
		sconn.Read(make([]byte, 0))
		fmt.Println("Finished read on first server")
		/*
			buf := bufio.NewReader(sconn)
			_, err := http.ReadRequest(buf)
			if err != nil {
				return
			}

			res := http.Response{
				Status: "200 OK",
			}
			res.Write(sconn)
		*/
	}

	go func() {
		for {
			sconn, err := hl.Accept()
			//_, err := hl.Accept()
			//defer sconn.Close()
			assert.NoError(t, err)
			go handleConnection(sconn)

		}
	}()

	// Now try to get an actual response from microsoft.com.
	fmt.Printf("trying for real...\n")
	cfg := &tls.Config{
		//	ServerName: "microsoft.com",
		ServerName: dst,
	}
	conn, err := tls.Dial("tcp", l.Addr().String(), cfg)
	assert.NoError(t, err)

	conn.Handshake()
	req, _ := http.NewRequest("get", "https://microsoft.com", nil)

	fmt.Printf("Writing request...\n")
	err = req.Write(conn)

	assert.NoError(t, err)

	res, err := http.ReadResponse(bufio.NewReader(conn), req)
	assert.NoError(t, err)
	fmt.Printf("Response: %#v", res)
	conn.Close()
	//time.Sleep(5 * time.Second)
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

// wrap wraps the specified listener in our default TLS listener.
func runDestinationServer() string {

	//l, err := net.Listen("tcp", ":0")
	l, err := tlsdefaults.Listen(":0", "../test/data/server.key", "../test/data/server.crt")
	//cfg, err := tlsdefaults.BuildListenerConfig(l.Addr().String(), "../test/data/server.key", "../test/data/server.crt")
	if err != nil {
		return ""
	}
	// This is a bit of a hack to make pre-shared TLS sessions work with uTLS. Ideally we'll make this
	// work with TLS 1.3, see https://github.com/getlantern/lantern-internal/issues/3057.
	//cfg.MaxVersion = tls.VersionTLS12

	log := golog.LoggerFor("lantern-proxy-tlslistener")

	handleConnection := func(sconn net.Conn) {
		log.Debug("Handling destination connection")
		//sconn.Handshake()

		//log.Debug("Finished handshake")
		//sconn.Read(make([]byte, 0))
		buf := bufio.NewReader(sconn)
		req, err := http.ReadRequest(buf)
		log.Debugf("Read request on destination %#v", req)
		if err != nil {
			log.Errorf("Read request error: %v", err)
			return
		}

		res := http.Response{
			Status: "200 OK",
		}
		res.Write(sconn)
		log.Debug("Wrote response")
	}

	go func() {
		for {
			conn, _ := l.Accept()
			log.Debug("Accepted connection on destination")
			//tlsConn := tls.Server(conn, cfg)
			//tlsConn := utls.NewServerConn(conn)
			go handleConnection(conn)

		}
	}()
	return l.Addr().String()
}
