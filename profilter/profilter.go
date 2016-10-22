// Lantern Pro middleware will identify Pro users and forward their requests
// immediately.  It will intercept non-Pro users and limit their total transfer

package profilter

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"github.com/getlantern/errors"
	"github.com/getlantern/golog"
	"github.com/getlantern/http-proxy-lantern/common"
	"github.com/getlantern/http-proxy/filters"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var (
	log = golog.LoggerFor("profilter")
)

type lanternProFilter struct {
	Opts *Opts
}

type token struct {
	userID     string
	clientIP   string
	serverIP   string
	expiration time.Time
}

type Opts struct {
	// ServerIP is the ip address of the server against which we'll check tokens.
	ServerIP string

	// PublicKey is the public key against which we check token signatures.
	PublicKey *rsa.PublicKey
}

func New(opts *Opts) filters.Filter {
	if opts.PublicKey == nil {
		panic("No PublicKey provided!")
	}
	if opts.ServerIP == "" {
		panic("No ServerIP provided!")
	}

	f := &lanternProFilter{
		Opts: opts,
	}

	return f
}

func (f *lanternProFilter) Apply(w http.ResponseWriter, req *http.Request, next filters.Next) error {
	tokenWithSignature := req.Header.Get(common.ProTokenHeader)
	if tokenWithSignature == "" {
		log.Debugf("Request missing %v header", common.ProTokenHeader)
		w.WriteHeader(http.StatusUnauthorized)
		return filters.Stop()
	}

	token, err := f.tokenFrom(tokenWithSignature)
	if err != nil {
		log.Error(err)
		w.WriteHeader(http.StatusForbidden)
		return filters.Stop()
	}

	if token.expiration.Before(time.Now()) {
		log.Errorf("Token expired at %v", token.expiration)
		w.WriteHeader(http.StatusForbidden)
		return filters.Stop()
	}

	clientIP, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		log.Errorf("Unable to split host and port for remote addr %v: %v", req.RemoteAddr, err)
		w.WriteHeader(http.StatusInternalServerError)
		return filters.Stop()
	}
	if clientIP != token.clientIP {
		log.Errorf("Client IP '%v' did not match token '%v'", clientIP, token.clientIP)
		w.WriteHeader(http.StatusForbidden)
		return filters.Stop()
	}

	if f.Opts.ServerIP != token.serverIP {
		log.Errorf("Server IP '%v' did not match token '%v'", req.RemoteAddr, token.serverIP)
		w.WriteHeader(http.StatusForbidden)
		return filters.Stop()
	}

	return next()
}

func (f *lanternProFilter) tokenFrom(tokenWithSignature string) (*token, error) {
	parts := strings.Split(tokenWithSignature, "|")
	if len(parts) != 2 {
		return nil, errors.New("Invalid token format")
	}
	tokenParts := strings.Split(parts[0], ",")
	if len(tokenParts) != 3 {
		return nil, errors.New("Invalid token format")
	}
	expiration, err := strconv.ParseInt(tokenParts[2], 10, 64)
	if err != nil {
		return nil, errors.New("Invalid token format, bad expiration %v: %v", tokenParts[3], err)
	}
	sig, err := base64.URLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, errors.New("Invalid token format, bad signature: %v", err)
	}
	hashed := sha256.Sum256([]byte(parts[0]))
	verifyErr := rsa.VerifyPKCS1v15(f.Opts.PublicKey, crypto.SHA256, hashed[:], sig)
	if verifyErr != nil {
		return nil, errors.New("Token failed signature verification: %v", verifyErr)
	}
	return &token{
		clientIP:   tokenParts[0],
		serverIP:   tokenParts[1],
		expiration: time.Unix(expiration, 0),
	}, nil
}
