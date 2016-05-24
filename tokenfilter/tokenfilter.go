package tokenfilter

import (
	"net/http"
	"net/http/httputil"
	"strings"

	"github.com/getlantern/golog"
	"github.com/getlantern/ops"

	"github.com/getlantern/http-proxy-lantern/common"
	"github.com/getlantern/http-proxy-lantern/mimic"
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
	op := ops.Enter("tokenfilter")
	defer op.Exit()

	if log.IsTraceEnabled() {
		reqStr, _ := httputil.DumpRequest(req, true)
		log.Tracef("Token Filter Middleware received request:\n%s", reqStr)
	}

	if f.token == "" {
		log.Debug("Not checking token")
		f.next.ServeHTTP(w, req)
		return
	}

	tokens := req.Header[common.TokenHeader]
	if tokens == nil || len(tokens) == 0 || tokens[0] == "" {
		log.Error(op.Errorf("No token provided, mimicking apache"))
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
			req.Header.Del(common.TokenHeader)
			log.Debugf("Allowing connection from %v to %v", req.RemoteAddr, req.Host)
			f.next.ServeHTTP(w, req)
		} else {
			log.Error(op.Errorf("Mismatched token(s) %v, mimicking apache", strings.Join(tokens, ",")))
			mimic.MimicApache(w, req)
		}
	}
}
