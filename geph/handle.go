package geph

import (
	"crypto/ed25519"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/rlp"
	"github.com/geph-official/geph2/libs/backedtcp"
	"github.com/geph-official/geph2/libs/cwl"
	"github.com/geph-official/geph2/libs/tinyss"
	"github.com/hashicorp/yamux"
	"github.com/patrickmn/go-cache"
	"github.com/xtaci/smux"
	"golang.org/x/time/rate"
)

// blacklist of local networks
var cidrBlacklist []*net.IPNet
var infiniteLimit = rate.NewLimiter(rate.Inf, 1000)

// session counter by using a forgetful cache. counting the elements gives us the session count.
var freeSessCounter = cache.New(time.Minute*10, time.Second*10)
var paidSessCounter = cache.New(time.Minute*10, time.Second*10)
var ipcache = cache.New(time.Hour, time.Hour)

func init() {
	for _, s := range []string{
		"127.0.0.1/8",
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"::1/128",
	} {
		_, n, _ := net.ParseCIDR(s)
		cidrBlacklist = append(cidrBlacklist, n)
	}
}

func isBlack(addr *net.TCPAddr) bool {
	for _, n := range cidrBlacklist {
		if n.Contains(addr.IP) {
			return true
		}
	}
	return false
}

var tunnCount uint64

func handle(rawClient net.Conn) {
	log.Debugf("handle called with", rawClient.RemoteAddr())
	rawClient.SetDeadline(time.Now().Add(time.Second * 30))
	tssClient, err := tinyss.Handshake(rawClient, 0)
	if err != nil {
		rawClient.Close()
		return
	}
	log.Debugf("tssClient with prot", tssClient.NextProt())
	// HACK: it's bridged if the remote address has a dot in it
	//isBridged := strings.Contains(rawClient.RemoteAddr().String(), ".")
	// sign the shared secret
	ssSignature := ed25519.Sign(seckey, tssClient.SharedSec())
	rlp.Encode(tssClient, &ssSignature)
	var limiter *rate.Limiter
	limiter = infiniteLimit
	slowLimit := false
	// "generic" stuff
	var acceptStream func() (net.Conn, error)
	rawClient.SetDeadline(time.Now().Add(time.Hour * 24))
	switch tssClient.NextProt() {
	case 0:
		defer tssClient.Close()
		// create smux context
		muxSrv, err := smux.Server(tssClient, &smux.Config{
			Version:           1,
			KeepAliveInterval: time.Minute * 10,
			KeepAliveTimeout:  time.Minute * 40,
			MaxFrameSize:      8192,
			MaxReceiveBuffer:  100 * 1024 * 1024,
			MaxStreamBuffer:   10 * 1024 * 1024,
		})
		if err != nil {
			log.Debugf("Error negotiating smux from", rawClient.RemoteAddr(), err)
			return
		}
		acceptStream = func() (n net.Conn, e error) {
			n, e = muxSrv.AcceptStream()
			return
		}
	case 2:
		defer tssClient.Close()
		// create smux context
		muxSrv, err := smux.Server(tssClient, &smux.Config{
			Version:           2,
			KeepAliveInterval: time.Minute * 2,
			KeepAliveTimeout:  time.Minute * 20,
			MaxFrameSize:      32768,
			MaxReceiveBuffer:  100 * 1024 * 1024,
			MaxStreamBuffer:   100 * 1024 * 1024,
		})
		if err != nil {
			log.Debugf("Error negotiating smux from", rawClient.RemoteAddr(), err)
			return
		}
		acceptStream = func() (n net.Conn, e error) {
			n, e = muxSrv.AcceptStream()
			return
		}
	case 'S':
		defer tssClient.Close()
		// create smux context
		muxSrv, err := yamux.Server(tssClient, &yamux.Config{
			AcceptBacklog:          1000,
			EnableKeepAlive:        false,
			KeepAliveInterval:      time.Hour,
			ConnectionWriteTimeout: time.Minute * 30,
			MaxStreamWindowSize:    100 * 1024 * 1024,
			LogOutput:              ioutil.Discard,
		})
		if err != nil {
			log.Debugf("Error negotiating yamux from", rawClient.RemoteAddr(), err)
			return
		}
		acceptStream = func() (n net.Conn, e error) {
			n, e = muxSrv.AcceptStream()
			return
		}
	case 'R':
		err = handleResumable(slowLimit, tssClient)
		log.Debugf("handleResumable returned with", err)
		if err != nil {
			tssClient.Close()
		}
		return
	}
	if slowLimit {
		limiter = rate.NewLimiter(100*1000, 1000*1000)
	}
	smuxLoop(fmt.Sprintf("%v", strings.Split(tssClient.RemoteAddr().String(), ":")[0]), limiter, acceptStream)
}

type scEntry struct {
	newConns chan net.Conn
	currConn net.Conn
	handle   *backedtcp.Socket
}

var sessionCache = make(map[[32]byte]*scEntry)
var sessionCacheLock sync.Mutex

func handleResumable(slowLimit bool, tssClient net.Conn) (err error) {
	log.Debugf("handling resumable from", tssClient.RemoteAddr())
	tssClient.SetDeadline(time.Now().Add(time.Second * 10))
	var clientHello struct {
		MetaSess [32]byte
		SessID   [32]byte
	}
	err = binary.Read(tssClient, binary.BigEndian, &clientHello)
	if err != nil {
		return
	}
	log.Debugf("[%v] M=%x, S=%x", tssClient.RemoteAddr(), clientHello.MetaSess, clientHello.SessID)
	sessionCacheLock.Lock()
	defer sessionCacheLock.Unlock()
	if bt, ok := sessionCache[clientHello.SessID]; ok {
		log.Debugf("[%v] found session", tssClient.RemoteAddr())
		bt.currConn.Close()
		bt.currConn = tssClient
		select {
		case bt.newConns <- tssClient:
			tssClient.Write([]byte{1})
		case <-time.After(time.Millisecond * 100):
			log.Debugf("******** somehow stuck **********")
		}
		return
	}
	log.Debugf("[%v] creating session", tssClient.RemoteAddr())
	tssClient.Write([]byte{0})
	ch := make(chan net.Conn, 1)
	ch <- tssClient
	btcp := backedtcp.NewSocket(func() (net.Conn, error) {
		select {
		case c := <-ch:
			return c, nil
		case <-time.After(time.Minute * 30):
			return nil, errors.New("timeout")
		}
	})
	sessionCache[clientHello.SessID] = &scEntry{
		newConns: ch,
		handle:   btcp,
		currConn: tssClient,
	}
	go func() {
		defer func() {
			sessionCacheLock.Lock()
			defer sessionCacheLock.Unlock()
			log.Debugf("deleting sessid %v", clientHello.SessID)
			delete(sessionCache, clientHello.SessID)
		}()
		defer btcp.Close()
		muxSrv, err := smux.Server(btcp, &smux.Config{
			Version:           2,
			KeepAliveInterval: time.Minute * 20,
			KeepAliveTimeout:  time.Minute * 40,
			MaxFrameSize:      32768,
			MaxReceiveBuffer:  1 * 1024 * 1024,
			MaxStreamBuffer:   256 * 1024,
		})
		if err != nil {
			return
		}
		acceptStream := func() (n net.Conn, e error) {
			n, e = muxSrv.AcceptStream()
			return
		}
		var limiter *rate.Limiter
		if slowLimit {
			limiter = slowLimitFactory.getLimiter(clientHello.MetaSess)
		} else {
			limiter = infiniteLimit
		}
		smuxLoop(fmt.Sprintf("%x", clientHello.MetaSess), limiter, acceptStream)
	}()
	return
}

func smuxLoop(sessid string, limiter *rate.Limiter, acceptStream func() (n net.Conn, e error)) {
	// copy the streams while
	for {
		soxclient, err := acceptStream()
		if err != nil {
			log.Debugf("failed accept stream", err)
			return
		}
		if limiter == infiniteLimit {
			paidSessCounter.SetDefault(sessid, true)
		} else {
			freeSessCounter.SetDefault(sessid, true)
		}
		go func() {
			defer soxclient.Close()
			soxclient.SetDeadline(time.Now().Add(time.Minute))
			var command []string
			err = rlp.Decode(&io.LimitedReader{R: soxclient, N: 1000}, &command)
			if err != nil {
				return
			}
			if len(command) == 0 {
				return
			}
			soxclient.SetDeadline(time.Time{})
			atomic.LoadUint64(&tunnCount)
			timeout := time.Minute * 30
			//log.Debugf("<%v> [%v] cmd %v", tc, timeout, command)
			// match command
			switch command[0] {
			case "proxy":
				if len(command) < 1 {
					return
				}
				rlp.Encode(soxclient, true)
				host := command[1]
				var remote net.Conn
				for _, ntype := range []string{"tcp6", "tcp4"} {
					tcpAddr, err := net.ResolveTCPAddr(ntype, host)
					if err != nil || isBlack(tcpAddr) {
						continue
					}
					remote, err = net.DialTimeout(ntype, tcpAddr.String(), time.Second*30)
					if err != nil {
						continue
					}
					break
				}
				if remote == nil {
					return
				}
				atomic.AddUint64(&tunnCount, 1)
				defer atomic.AddUint64(&tunnCount, ^uint64(0))
				defer remote.Close()
				go func() {
					defer remote.Close()
					defer soxclient.Close()
					cwl.CopyWithLimit(remote, soxclient, limiter, nil, timeout)
				}()
				cwl.CopyWithLimit(soxclient, remote, limiter, nil, timeout)
			case "ip":
				var ip string
				if ipi, ok := ipcache.Get("ip"); ok {
					ip = ipi.(string)
				} else {
					addr := "http://checkip.amazonaws.com"
					resp, err := http.Get(addr)
					if err != nil {
						return
					}
					defer resp.Body.Close()
					ipb, err := ioutil.ReadAll(resp.Body)
					if err != nil {
						return
					}
					ip = string(ipb)
					ipcache.SetDefault("ip", ip)
				}
				rlp.Encode(soxclient, true)
				rlp.Encode(soxclient, ip)
				time.Sleep(time.Second)
			}
		}()
	}
}
