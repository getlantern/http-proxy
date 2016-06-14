// Lantern Pro middleware will identify Pro users and forward their requests
// immediately.  It will intercept non-Pro users and limit their total transfer

package profilter

import (
	"net/http"
	"net/http/httputil"
	"sync"
	"sync/atomic"

	"github.com/getlantern/golog"

	"github.com/getlantern/http-proxy/filters"

	"github.com/getlantern/http-proxy-lantern/common"
	"github.com/getlantern/http-proxy-lantern/mimic"
	"github.com/getlantern/http-proxy-lantern/redis"
)

var log = golog.LoggerFor("profilter")

type TokensMap map[string]bool

type lanternProFilter struct {
	enabled   int32
	proTokens *atomic.Value
	// Tokens write-only mutex
	tkwMutex sync.Mutex
	proConf  *proConfig
}

type Options struct {
	RedisOpts *redis.Options
	ServerID  string
}

func New(opts *Options) (filters.Filter, error) {
	f := &lanternProFilter{
		proTokens: new(atomic.Value),
	}
	// atomic.Value can't be copied after Store has been called
	f.proTokens.Store(make(TokensMap))

	var err error
	f.proConf, err = NewRedisProConfig(opts.RedisOpts, opts.ServerID, f)
	if err != nil {
		return nil, err
	}

	err = f.proConf.Run(true)
	return f, nil
}

func (f *lanternProFilter) Apply(w http.ResponseWriter, req *http.Request, next filters.Next) error {
	lanternProToken := req.Header.Get(common.ProTokenHeader)

	if log.IsTraceEnabled() {
		reqStr, _ := httputil.DumpRequest(req, true)
		log.Tracef("Lantern Pro Filter Middleware received request:\n%s", reqStr)
		if lanternProToken != "" {
			log.Tracef("Lantern Pro Token found")
		}
	}

	req.Header.Del(common.ProTokenHeader)

	if !f.isEnabled() {
		return next()
	}

	// If a Pro token is found in the header, test if its valid and then let
	// the request pass.
	if lanternProToken != "" && f.tokenExists(lanternProToken) {
		return next()
	}

	log.Debugf("Mismatched Pro token %s from %s, mimicking apache", lanternProToken, req.RemoteAddr)
	mimic.MimicApache(w, req)
	return filters.Stop()
}

func (f *lanternProFilter) isEnabled() bool {
	return atomic.LoadInt32(&f.enabled) != 0
}

func (f *lanternProFilter) Enable() {
	atomic.StoreInt32(&f.enabled, 1)
}

func (f *lanternProFilter) Disable() {
	atomic.StoreInt32(&f.enabled, 0)
}

// AddTokens appends a series of tokens to the current list
// Note: this isn't used at the moment, just kept for documentation
// purposes.  The only difference with SetTokens is that appends
// instead of reset.  Since we use a copy-on-write sync approach
// there is no advantage to this vs. just resetting, unless we
// can safely assume that user tokens are never updated
func (f *lanternProFilter) AddTokens(tokens ...string) {
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
func (f *lanternProFilter) SetTokens(tokens ...string) {
	// Copy-on-write.  Writes are far less common than reads.
	f.tkwMutex.Lock()
	defer f.tkwMutex.Unlock()

	tks := make(TokensMap)
	for _, t := range tokens {
		tks[t] = true
	}
	f.proTokens.Store(tks)
}

func (f *lanternProFilter) ClearTokens() {
	// Synchronize with writers
	f.tkwMutex.Lock()
	defer f.tkwMutex.Unlock()

	f.proTokens.Store(make(TokensMap))
}

func (f *lanternProFilter) tokenExists(token string) bool {
	tks := f.proTokens.Load().(TokensMap)
	_, ok := tks[token]
	return ok
}
