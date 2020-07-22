// Package geph provides a listener that emulates geph
package geph

import (
	"encoding/hex"
	"io/ioutil"
	"net"
	"time"

	"github.com/ethereum/go-ethereum/rlp"
	"github.com/getlantern/golog"
	"golang.org/x/crypto/ed25519"

	"github.com/geph-official/geph2/libs/cshirt2"
	"github.com/geph-official/geph2/libs/tinyss"
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
	rawClient.SetDeadline(time.Now().Add(time.Second * 10))
	client, err := cshirt2.Server(pubkey, false, rawClient)
	if err != nil {
		rawClient.Close()
		return nil, err
	}
	rawClient.SetDeadline(time.Now().Add(time.Hour * 24))
	client.SetDeadline(time.Now().Add(time.Second * 30))
	tssClient, err := tinyss.Handshake(client, 0 /*nextProt*/)
	if err != nil {
		client.Close()
		return nil, err
	}
	// HACK: it's bridged if the remote address has a dot in it
	//isBridged := strings.Contains(client.RemoteAddr().String(), ".")
	// sign the shared secret
	ssSignature := ed25519.Sign(seckey, tssClient.SharedSec())
	rlp.Encode(tssClient, &ssSignature)
	client.SetDeadline(time.Now().Add(time.Hour * 24))
	return tssClient, nil
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
		_, key, _ := ed25519.GenerateKey(nil)
		ioutil.WriteFile(keyfile, key, 0600)
		goto retry
	}
	seckey = bts
	pubkey = seckey.Public().(ed25519.PublicKey)
	ioutil.WriteFile(keyfile+".pub", []byte(hex.EncodeToString(pubkey)), 0600)
}
