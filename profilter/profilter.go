// Lantern Pro middleware will identify Pro users and forward their requests
// immediately.  It will intercept non-Pro users and limit their total transfer

package profilter

import (
	"net/http"
	"net/http/httputil"
	"sync"
	"sync/atomic"

	"github.com/getlantern/golog"
	"github.com/gorilla/context"

	"github.com/getlantern/http-proxy/filters"
	"github.com/getlantern/http-proxy/listeners"

	"github.com/getlantern/http-proxy-lantern/common"
	throttle "github.com/getlantern/http-proxy-lantern/listeners"
	redislib "gopkg.in/redis.v3"
)

var log = golog.LoggerFor("profilter")

type TokensMap map[string]bool

type lanternProFilter struct {
	enabled   int32
	proTokens *atomic.Value
	// Tokens write-only mutex
	tkwMutex            sync.Mutex
	proConf             *proConfig
	keepProTokenDomains []string
}

type Options struct {
	RedisClient         *redislib.Client
	ServerID            string
	KeepProTokenDomains []string
}

func New(opts *Options) (filters.Filter, error) {
	f := &lanternProFilter{
		proTokens:           new(atomic.Value),
		keepProTokenDomains: opts.KeepProTokenDomains,
	}
	// atomic.Value can't be copied after Store has been called
	f.proTokens.Store(make(TokensMap))

	f.proConf = NewRedisProConfig(opts.RedisClient, opts.ServerID, f)

	err := f.proConf.Run(false)
	return f, err
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

	shouldDelete := true
	for _, domain := range f.keepProTokenDomains {
		if req.Host == domain {
			shouldDelete = false
			break
		}
	}
	if shouldDelete {
		req.Header.Del(common.ProTokenHeader)
	}

	if f.isEnabled() && lanternProToken != "" && f.tokenExists(lanternProToken) {
		wc := context.Get(req, "conn").(listeners.WrapConn)
		wc.ControlMessage("throttle", throttle.Never)
	}

	return next()
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
