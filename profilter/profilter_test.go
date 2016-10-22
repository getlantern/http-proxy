package profilter

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"github.com/getlantern/http-proxy-lantern/common"
	"github.com/getlantern/http-proxy/filters"
	"github.com/getlantern/httptest"
	"github.com/getlantern/keyman"
	"github.com/stretchr/testify/assert"
	"net/http"
	"strings"
	"testing"
	"time"
)

var (
	clientIP = "66.69.23.12"
	serverIP = "188.152.56.23"

	pk *rsa.PrivateKey
	f  filters.Filter
)

func init() {
	_pk, err := keyman.GeneratePK(4096)
	if err != nil {
		panic(err)
	}
	pk = _pk.RSA()
	f = New(&Opts{
		ServerIP:  serverIP,
		PublicKey: &pk.PublicKey,
	})
}

func TestOK(t *testing.T) {
	expiration := time.Now().Add(15 * time.Minute)
	token := strings.Join([]string{clientIP, serverIP, fmt.Sprint(expiration.Unix())}, ",")
	hashed := sha256.Sum256([]byte(token))
	sig, err := rsa.SignPKCS1v15(rand.Reader, pk, crypto.SHA256, hashed[:])
	if !assert.NoError(t, err) {
		return
	}
	tokenWithSignature := fmt.Sprintf("%v|%v", token, base64.URLEncoding.EncodeToString(sig))
	req, _ := http.NewRequest("GET", "https://www.google.com", nil)
	req.RemoteAddr = fmt.Sprintf("%v:16234", clientIP)
	req.Header.Set(common.ProTokenHeader, tokenWithSignature)
	resp := httptest.NewRecorder(nil)

	passed := false
	applyErr := f.Apply(resp, req, func() error {
		resp.WriteHeader(http.StatusOK)
		passed = true
		return nil
	})

	assert.NoError(t, applyErr)
	assert.True(t, passed)
	assert.Equal(t, http.StatusOK, resp.Result().StatusCode)
}
