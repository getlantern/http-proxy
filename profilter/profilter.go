// Lantern Pro middleware will identify Pro users and forward their requests
// immediately.  It will intercept non-Pro users and limit their total transfer

package profilter

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"sync"
	"sync/atomic"

	"github.com/getlantern/golog"
	"github.com/getlantern/http-proxy-lantern/mimic"

	"github.com/getlantern/http-proxy-lantern/common"
)

var log = golog.LoggerFor("profilter")

type TokensMap map[string]bool

type LanternProFilter struct {
	next      http.Handler
	enabled   int32
	proTokens *atomic.Value
	// Tokens write-only mutex
	tkwMutex sync.Mutex
	proConf  *proConfig
}

type optSetter func(f *LanternProFilter) error

func RedisConfigSetter(redisAddr, serverId string) optSetter {
	return func(f *LanternProFilter) (err error) {
		if f.proConf, err = NewRedisProConfig(redisAddr, serverId, f); err != nil {
			return
		}

		isPro, err := f.proConf.IsPro()
		if err != nil {
			return fmt.Errorf("Error reading assigned users in bootstrapping: %s", err)
		}
		// Currently, we run a Pro server starting as a free server
		// that can turn into a Pro server.  This is an initial design,
		// that will change if we decide to use separate queues, and therefore
		// can configure the proxy at launch time
		if err := f.proConf.Run(isPro); err != nil {
			return fmt.Errorf("Error configuring Pro: %s", err)
		}
		return
	}
}

func New(next http.Handler, setters ...optSetter) (*LanternProFilter, error) {
	f := &LanternProFilter{
		next:      next,
		enabled:   0,
		proTokens: new(atomic.Value),
	}

	// atomic.Value can't be copied after Store has been called
	f.proTokens.Store(make(TokensMap))

	for _, s := range setters {
		if err := s(f); err != nil {
			return nil, err
		}
	}
	return f, nil
}

func (f *LanternProFilter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	lanternProToken := req.Header.Get(common.ProTokenHeader)

	if log.IsTraceEnabled() {
		reqStr, _ := httputil.DumpRequest(req, true)
		log.Tracef("Lantern Pro Filter Middleware received request:\n%s", reqStr)
		if lanternProToken != "" {
			log.Tracef("Lantern Pro Token found")
		}
	}

	req.Header.Del(common.ProTokenHeader)

	if f.isEnabled() {
		// If a Pro token is found in the header, test if its valid and then let
		// the request pass.
		if lanternProToken != "" && f.tokenExists(lanternProToken) {
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

// AddTokens appends a series of tokens to the current list
// Note: this isn't used at the moment, just kept for documentation
// purposes.  The only difference with SetTokens is that appends
// instead of reset.  Since we use a copy-on-write sync approach
// there is no advantage to this vs. just resetting, unless we
// can safely assume that user tokens are never updated
func (f *LanternProFilter) AddTokens(tokens ...string) {
	// Copy-on-write.  Writes are far less common than reads.
	f.tkwMutex.Lock()
	defer f.tkwMutex.Unlock()

	tks1 := f.proTokens.Load().(TokensMap)
	tks2 := make(TokensMap)
	for k, _ := range tks1 {
		tks2[k] = true
	}
	for _, t := range tokens {
		tks2[t] = true
	}
	f.proTokens.Store(tks2)
}

// SetTokens sets the tokens to the provided list, removing
// any previous token
func (f *LanternProFilter) SetTokens(tokens ...string) {
	// Copy-on-write.  Writes are far less common than reads.
	f.tkwMutex.Lock()
	defer f.tkwMutex.Unlock()

	tks := make(TokensMap)
	for _, t := range tokens {
		tks[t] = true
	}
	f.proTokens.Store(tks)
}

func (f *LanternProFilter) ClearTokens() {
	// Synchronize with writers
	f.tkwMutex.Lock()
	defer f.tkwMutex.Unlock()

	f.proTokens.Store(make(TokensMap))
}

func (f *LanternProFilter) tokenExists(token string) bool {
	tks := f.proTokens.Load().(TokensMap)
	_, ok := tks[token]
	return ok
}
