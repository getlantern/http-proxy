package tokenfilter

import (
	"net/http"
	"net/http/httputil"

	"github.com/getlantern/golog"
	"github.com/getlantern/http-proxy-lantern/mimic"
)

const (
	tokenHeader = "X-Lantern-Auth-Token"
)

var log = golog.LoggerFor("tokenfilter")

type TokenFilter struct {
	next  http.Handler
	token string
}

type optSetter func(f *TokenFilter) error

func TokenSetter(token string) optSetter {
	return func(f *TokenFilter) error {
		f.token = token
		return nil
	}
}

func New(next http.Handler, setters ...optSetter) (*TokenFilter, error) {
	f := &TokenFilter{
		next:  next,
		token: "",
	}
	for _, s := range setters {
		if err := s(f); err != nil {
			return nil, err
		}
	}

	return f, nil
}

func (f *TokenFilter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if log.IsTraceEnabled() {
		reqStr, _ := httputil.DumpRequest(req, true)
		log.Tracef("Token Filter Middleware received request:\n%s", reqStr)
	}

	token := req.Header.Get(tokenHeader)
	req.Header.Del(tokenHeader)
	if f.token == "" {
		f.next.ServeHTTP(w, req)
		return
	}
	switch token {
	case f.token:
		f.next.ServeHTTP(w, req)
	case "":
		log.Debugf("No token provided from %s, mimicking apache", req.RemoteAddr)
		mimic.MimicApache(w, req)
	default:
		log.Debugf("Mismatched token %s from %s, mimicking apache", token, req.RemoteAddr)
		mimic.MimicApache(w, req)
	}
}
