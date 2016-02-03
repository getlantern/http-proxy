package tokenfilter

import (
	"net/http"
	"net/http/httputil"
	"strings"

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

	if f.token == "" {
		f.next.ServeHTTP(w, req)
		return
	}

	tokens := req.Header[tokenHeader]
	if tokens == nil || len(tokens) == 0 || tokens[0] == "" {
		log.Debugf("No token provided from %s, mimicking apache", req.RemoteAddr)
		mimic.MimicApache(w, req)
	} else {
		tokenMatched := false
		for _, candidate := range tokens {
			if candidate == f.token {
				tokenMatched = true
				break
			}
		}
		if tokenMatched {
			req.Header.Del(tokenHeader)
			f.next.ServeHTTP(w, req)
		} else {
			log.Debugf("Mismatched token(s) %s from %s, mimicking apache", strings.Join(tokens, ","), req.RemoteAddr)
		}
		mimic.MimicApache(w, req)
	}
}
