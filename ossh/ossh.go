package ossh

import (
	"crypto/x509"
	"encoding/pem"
	"io/ioutil"
	"net"

	"github.com/getlantern/errors"
	"github.com/getlantern/golog"
	"github.com/getlantern/ossh"
	"golang.org/x/crypto/ssh"
)

var log = golog.LoggerFor("ossh-listener")

func Wrap(l net.Listener, obfuscationKeyword, hostKeyFile string) (net.Listener, error) {
	keyPEM, err := ioutil.ReadFile(hostKeyFile)
	if err != nil {
		return nil, errors.New("failed to read host key file: %v", err)
	}
	keyBlock, rest := pem.Decode(keyPEM)
	if len(rest) > 0 {
		return nil, errors.New("failed to decode host key as PEM block")
	}
	if keyBlock.Type != "RSA PRIVATE KEY" {
		return nil, errors.New("expected key block of type 'RSA PRIVATE KEY', but got %v", keyBlock.Type)
	}
	rsaKey, err := x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
	if err != nil {
		return nil, errors.New("failed to parse host key as PKCS1: %v", err)
	}
	sshKey, err := ssh.NewSignerFromKey(rsaKey)
	if err != nil {
		return nil, errors.New("failed to convert RSA key to SSH key: %v", err)
	}
	cfg := ossh.ListenerConfig{
		HostKey:            sshKey,
		ObfuscationKeyword: obfuscationKeyword,
		Logger: func(_ string, err error, _ map[string]interface{}) {
			if err != nil {
				log.Errorf("error from OSSH logger: %v", err)
			}
		},
	}
	return ossh.WrapListener(l, cfg)
}
