package tlslistener

import (
	"bufio"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/getlantern/golog"
	tls "github.com/getlantern/utls"
)

func TestAbortOnHello(t *testing.T) {
	disallowLookbackForTesting = true
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
		for {
			conn, err := hl.Accept()
			time.Sleep(1 * time.Second)
			assert.NoError(t, err)
			go handleConnection(conn)
		}
	}()

	cfg := &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         "microsoft.com",
	}

	_, err = tls.Dial("tcp", l.Addr().String(), cfg)
	assert.Error(t, err)

	// Now let's make a real connection directly to Microsoft to get a session
	// to resume. Then let's try to resume it.
	cfg = &tls.Config{
		//InsecureSkipVerify: true,
		ServerName:         "microsoft.com",
		ClientSessionCache: newLocalCache(),
	}

	log := golog.LoggerFor("hello-test")
	log.Debugf("cache: %#v", cfg.ClientSessionCache)
	tlsConn, err := tls.Dial("tcp", "microsoft.com:443", cfg)
	assert.NoError(t, err)
	get, err := http.NewRequest("GET", "/", nil)
	assert.NoError(t, err)
	err = get.Write(tlsConn)
	assert.NoError(t, err)
	tlsConn.Close()

	log.Debugf("cache: %#v", cfg.ClientSessionCache)

	/*
		rawConn, err := net.DialTimeout("tcp", l.Addr().String(), 4*time.Second)
		assert.NoError(t, err)

		//clientSessionCache := tls.NewLRUClientSessionCache(5)

		ss := &tls.ClientSessionState{}
		ss.SetSessionTicket([]uint8{6})
		ss.SetVers(tls.VersionTLS12)
		//ss.SetCipherSuite(sss.CipherSuite)
		//ss.SetMasterSecret(sss.MasterSecret)

		cfg = &tls.Config{
			//InsecureSkipVerify: true,
		}
		uconn := tls.UClient(rawConn, cfg, tls.HelloChrome_Auto)
		uconn.SetSessionState(ss)

		handshakeErr := uconn.Handshake()
		assert.NoError(t, handshakeErr)
	*/
}

type localCache struct {
	cache tls.ClientSessionCache
	log   golog.Logger
}

func (lc *localCache) Get(sessionKey string) (session *tls.ClientSessionState, ok bool) {
	return lc.cache.Get(sessionKey)
}

// Put adds the ClientSessionState to the cache with the given key. It might
// get called multiple times in a connection if a TLS 1.3 server provides
// more than one session ticket. If called with a nil *ClientSessionState,
// it should remove the cache entry.
func (lc *localCache) Put(sessionKey string, cs *tls.ClientSessionState) {
	lc.log.Debugf("Putting into cache: %v", sessionKey)
	lc.cache.Put(sessionKey, cs)
}

func newLocalCache() tls.ClientSessionCache {
	return &localCache{
		cache: tls.NewLRUClientSessionCache(1),
		log:   golog.LoggerFor("hello-test-cache"),
	}
}
