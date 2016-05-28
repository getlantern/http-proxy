package tokenfilter

import (
	"net/http"
	"net/http/httputil"
	"strings"

	"github.com/getlantern/golog"

	"github.com/getlantern/http-proxy/filter"

	"github.com/getlantern/http-proxy-lantern/common"
	"github.com/getlantern/http-proxy-lantern/mimic"
)

var log = golog.LoggerFor("tokenfilter")

type tokenFilter struct {
	token string
}

func New(token string) filter.Filter {
	return &tokenFilter{
		token: token,
	}
}

func (f *tokenFilter) Apply(w http.ResponseWriter, req *http.Request) (bool, error, string) {
	if log.IsTraceEnabled() {
		reqStr, _ := httputil.DumpRequest(req, true)
		log.Tracef("Token Filter Middleware received request:\n%s", reqStr)
	}

	if f.token == "" {
		return filter.Continue()
	}

	tokens := req.Header[common.TokenHeader]
	if tokens == nil || len(tokens) == 0 || tokens[0] == "" {
		log.Debugf("No token provided from %s for request to %v, mimicking apache", req.RemoteAddr, req.Host)
		mimic.MimicApache(w, req)
		return filter.Stop()
	}
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
		return filter.Continue()
	}
	log.Debugf("Mismatched token(s) %s from %s for request to %v, mimicking apache", strings.Join(tokens, ","), req.RemoteAddr, req.Host)
	mimic.MimicApache(w, req)
	return filter.Stop()
}
