package tlslistener

import (
	"bufio"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	tls "github.com/getlantern/utls"
)

func TestAbortOnHello(t *testing.T) {
	l, err := net.Listen("tcp", ":0")
	assert.NoError(t, err)

	hl, err := Wrap(l, "../test/data/server.key", "../test/data/server.crt", "dummytickets", true)
	assert.NoError(t, err)

	handleConnection := func(sconn net.Conn) {
		buf := bufio.NewReader(sconn)
		_, err := http.ReadRequest(buf)
		if err != nil {
			return
		}
		res := http.Response{
			Status: "200 OK",
		}
		res.Write(sconn)
	}

	go func() {
		sconn, err := hl.Accept()
		//defer sconn.Close()
		time.Sleep(2 * time.Second)
		assert.NoError(t, err)
		go handleConnection(sconn)
	}()

	cfg := &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         "microsoft.com",
	}

	_, err = tls.Dial("tcp", l.Addr().String(), cfg)
	assert.Error(t, err)
}
