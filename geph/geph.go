// Package geph provides a listener that emulates geph
package geph

import (
	"io/ioutil"
	"net"
	"time"

	"github.com/getlantern/golog"
	"golang.org/x/crypto/ed25519"

	"github.com/geph-official/geph2/libs/cshirt2"
)

var (
	log    = golog.LoggerFor("geph")
	pubkey ed25519.PublicKey
	seckey ed25519.PrivateKey
)

// Wrap wraps the given Listener into a Listener that applies the specified tos
// to all connections.
func Wrap(l net.Listener, keyfile string) net.Listener {
	loadKey(keyfile)
	return &gephListener{l}
}

type gephListener struct {
	wrapped net.Listener
}

func (l *gephListener) Accept() (net.Conn, error) {
	rawClient, err := l.wrapped.Accept()
	if err != nil {
		return nil, err
	}
	log.Debugf("Accepted from client at %v", rawClient.RemoteAddr())
	go func() {
		rawClient.SetDeadline(time.Now().Add(time.Second * 10))
		client, err := cshirt2.Server(pubkey, false, rawClient)
		if err != nil {
			rawClient.Close()
			return
		}
		rawClient.SetDeadline(time.Now().Add(time.Hour * 24))
		handle(client)
	}()
	return rawClient, nil
}

func (l *gephListener) Addr() net.Addr {
	return l.wrapped.Addr()
}

func (l *gephListener) Close() error {
	return l.wrapped.Close()
}

func loadKey(keyfile string) {
retry:
	bts, err := ioutil.ReadFile(keyfile)
	if err != nil {
		// genkey
		pubkey, key, _ := ed25519.GenerateKey(nil)
		ioutil.WriteFile(keyfile, key, 0600)
		ioutil.WriteFile(keyfile+".pub", pubkey, 0600)
		goto retry
	}
	seckey = bts
	pubkey = seckey.Public().(ed25519.PublicKey)
}
