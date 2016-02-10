// Lantern Pro middleware will identify Pro users and forward their requests
// immediately.  It will intercept non-Pro users and limit their total transfer

package profilter

import (
	"net/http"
	"net/http/httputil"

	"github.com/Workiva/go-datastructures/set"

	"github.com/getlantern/golog"
	"github.com/getlantern/http-proxy-lantern/mimic"
)

const (
	proTokenHeader = "X-Lantern-Pro-Token"
)

var log = golog.LoggerFor("profilter")

type LanternProFilter struct {
	next      http.Handler
	proTokens *set.Set
}

type optSetter func(f *LanternProFilter) error

func New(next http.Handler, setters ...optSetter) (*LanternProFilter, error) {
	f := &LanternProFilter{
		next:      next,
		proTokens: set.New(),
	}

	for _, s := range setters {
		if err := s(f); err != nil {
			return nil, err
		}
	}

	return f, nil
}

func (f *LanternProFilter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if log.IsTraceEnabled() {
		reqStr, _ := httputil.DumpRequest(req, true)
		log.Tracef("Lantern Pro Filter Middleware received request:\n%s", reqStr)
	}

	lanternProToken := req.Header.Get(proTokenHeader)
	req.Header.Del(proTokenHeader)
	if lanternProToken != "" {
		log.Tracef("Lantern Pro Token found")
	}

	// If a Pro token is found in the header, test if its valid and then let
	// the request pass.
	if lanternProToken != "" {
		if f.proTokens.Exists(lanternProToken) {
			f.next.ServeHTTP(w, req)
		} else {
			log.Debugf("Mismatched Pro token %s from %s, mimicking apache", lanternProToken, req.RemoteAddr)
			mimic.MimicApache(w, req)
		}
	} else {
		f.next.ServeHTTP(w, req)
	}
}
