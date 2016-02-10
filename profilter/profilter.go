// Lantern Pro middleware will identify Pro users and forward their requests
// immediately.  It will intercept non-Pro users and limit their total transfer

package profilter

import (
	"net/http"
	"net/http/httputil"
	"sync/atomic"

	"github.com/Workiva/go-datastructures/set"

	"github.com/getlantern/golog"
	"github.com/getlantern/http-proxy-lantern/mimic"

	"github.com/getlantern/http-proxy-lantern/common"
)

var log = golog.LoggerFor("profilter")

type LanternProFilter struct {
	next      http.Handler
	enabled   int32
	proTokens *set.Set
}

type optSetter func(f *LanternProFilter) error

func New(next http.Handler, setters ...optSetter) (*LanternProFilter, error) {
	f := &LanternProFilter{
		next:      next,
		enabled:   0,
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

	lanternProToken := req.Header.Get(common.ProTokenHeader)
	req.Header.Del(common.ProTokenHeader)
	if lanternProToken != "" {
		log.Tracef("Lantern Pro Token found")
	}

	if f.isEnabled() {
		// If a Pro token is found in the header, test if its valid and then let
		// the request pass.
		if lanternProToken != "" && f.proTokens.Exists(lanternProToken) {
			f.next.ServeHTTP(w, req)
		} else {
			log.Debugf("Mismatched Pro token %s from %s, mimicking apache", lanternProToken, req.RemoteAddr)
			mimic.MimicApache(w, req)
		}
	} else {
		f.next.ServeHTTP(w, req)
	}
}

func (f *LanternProFilter) isEnabled() bool {
	return atomic.LoadInt32(&f.enabled) != 0
}

func (f *LanternProFilter) Enable() {
	atomic.StoreInt32(&f.enabled, 1)
}

func (f *LanternProFilter) Disable() {
	atomic.StoreInt32(&f.enabled, 0)
}

func (f *LanternProFilter) UpdateTokens(tokens []string) {
	f.proTokens.Clear()
	f.proTokens.Add(tokens)
}
