package tokenfilter

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"strings"

	"github.com/getlantern/golog"
	"github.com/getlantern/ops"

	"github.com/getlantern/http-proxy/filters"

	"github.com/getlantern/http-proxy-lantern/common"
	"github.com/getlantern/http-proxy-lantern/mimic"
)

var log = golog.LoggerFor("tokenfilter")

type tokenFilter struct {
	token string
}

func New(token string) filters.Filter {
	return &tokenFilter{
		token: token,
	}
}

func (f *tokenFilter) Apply(w http.ResponseWriter, req *http.Request, next filters.Next) error {
	op := ops.Begin("tokenfilter")
	defer op.End()

	if log.IsTraceEnabled() {
		reqStr, _ := httputil.DumpRequest(req, true)
		log.Tracef("Token Filter Middleware received request:\n%s", reqStr)
	}

	if f.token == "" {
		log.Debug("Not checking token")
		return next()
	}

	tokens := req.Header[common.TokenHeader]
	if tokens == nil || len(tokens) == 0 || tokens[0] == "" {
		log.Error(errorf(op, "No token provided, mimicking apache"))
		mimic.MimicApache(w, req)
		return filters.Stop()
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
		return next()
	}
	log.Error(errorf(op, "Mismatched token(s) %v, mimicking apache", strings.Join(tokens, ",")))
	mimic.MimicApache(w, req)
	return filters.Stop()
}

func errorf(op ops.Op, msg string, args ...interface{}) error {
	return op.FailIf(fmt.Errorf(msg, args...))
}
