// Package banditcallback emits a best-effort per-device first-seen
// callback to the lantern-cloud bandit API so http-proxy arms get a
// reward signal without the client having to make the callback itself.
//
// The emitter holds a TTL-bounded map of device-ids it has already
// reported within the window. A device's first connection within the
// window triggers an async GET to the configured callback URL with the
// per-arm token (plumbed at provisioning) and the device-id; subsequent
// connections from the same device within TTL are suppressed. After the
// window, the next connection re-fires the callback. This matches the
// API-side dedup TTL so legitimate "device returns after a gap" events
// land, while a misbehaving client hammering the proxy can't pump the
// arm's reward signal.
package banditcallback

import (
	"context"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/getlantern/golog"
	"github.com/getlantern/proxy/v3/filters"
)

var log = golog.LoggerFor("banditcallback")

// Emitter fires per-device first-seen callbacks to the lantern-cloud
// API. Disabled when CallbackURL or Token is empty — the right
// default for non-bandit-eligible tracks.
type Emitter struct {
	token       string
	callbackURL string
	ttl         time.Duration
	client      *http.Client

	mu        sync.Mutex
	seen      map[string]time.Time
	lastSweep time.Time

	// Counters exposed for the proxy's metrics layer; readable via
	// the accessor methods. Bumped under mu since the proxy reads
	// them rarely (periodic flush) and bumps them on a hot path
	// (every device-id request); atomic.AddUint64 would add a CPU
	// barrier on every Apply for no observable win.
	emitted    uint64
	suppressed uint64
}

// New returns an emitter. token and callbackURL come from the daemon's
// INI (banditcallbacktoken / banditcallbackurl). ttl is the per-device
// dedup window — typically matches the API-side ProbeTTLForPollInterval
// at the daemon's expected poll. Zero ttl uses 10m as a sensible default.
func New(token, callbackURL string, ttl time.Duration) *Emitter {
	if ttl <= 0 {
		ttl = 10 * time.Minute
	}
	return &Emitter{
		token:       token,
		callbackURL: callbackURL,
		ttl:         ttl,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
		seen: make(map[string]time.Time),
	}
}

// Enabled reports whether the emitter will actually fire callbacks.
// Callers can use this to skip computing the device-id when the daemon
// wasn't provisioned with a token.
func (e *Emitter) Enabled() bool {
	return e != nil && e.token != "" && e.callbackURL != ""
}

// EmitIfFirstSeen records the device-id and, if it hasn't been seen
// within the TTL window, fires an async best-effort GET to the
// configured callback URL. Returns immediately — the HTTP request runs
// in its own goroutine and any failure is logged at debug only (the
// callback is a hint to the bandit, not a correctness signal).
func (e *Emitter) EmitIfFirstSeen(ctx context.Context, deviceID string) {
	if !e.Enabled() || deviceID == "" {
		return
	}

	now := time.Now()
	first := e.checkAndRecord(deviceID, now)
	if !first {
		return
	}

	go e.fire(ctx, deviceID)
}

// checkAndRecord returns true if the deviceID is first-seen within the
// TTL window. The map is swept opportunistically whenever the gap
// since the last sweep exceeds the TTL — bounding worst-case map size
// at ~2× the unique device-ids seen in one TTL window without a
// dedicated reaper goroutine.
func (e *Emitter) checkAndRecord(deviceID string, now time.Time) bool {
	e.mu.Lock()
	defer e.mu.Unlock()

	if now.Sub(e.lastSweep) > e.ttl {
		for k, t := range e.seen {
			if now.Sub(t) > e.ttl {
				delete(e.seen, k)
			}
		}
		e.lastSweep = now
	}

	if prev, ok := e.seen[deviceID]; ok && now.Sub(prev) <= e.ttl {
		e.suppressed++
		return false
	}
	e.seen[deviceID] = now
	e.emitted++
	return true
}

func (e *Emitter) fire(ctx context.Context, deviceID string) {
	// Detach from the request context so a closing client connection
	// doesn't cancel the outbound HTTP request. The callback is for
	// the bandit's benefit, not the client's; we want it to complete
	// even if the client just hung up.
	fireCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	u, err := url.Parse(e.callbackURL)
	if err != nil {
		log.Debugf("banditcallback: invalid callback URL %q: %v", e.callbackURL, err)
		return
	}
	q := u.Query()
	q.Set("token", e.token)
	q.Set("did", deviceID)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(fireCtx, http.MethodGet, u.String(), nil)
	if err != nil {
		log.Debugf("banditcallback: build request: %v", err)
		return
	}

	resp, err := e.client.Do(req)
	if err != nil {
		log.Debugf("banditcallback: request failed: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		log.Debugf("banditcallback: API returned %d for did %s", resp.StatusCode, deviceID)
	}
}

// Stats returns (emitted, suppressed) counter values. Cheap; takes the
// same mu the hot path uses but the hot path holds it for microseconds.
func (e *Emitter) Stats() (emitted, suppressed uint64) {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.emitted, e.suppressed
}

// Filter is a proxy filter that calls EmitIfFirstSeen on the device-id
// header. Wraps Emitter so the proxy can install it in its filter chain
// independently of devicefilter (which only runs for non-pro tracks).
// The filter is a near-noop hot path: under TTL, just a map read +
// counter bump under a brief mutex; misses kick off an async goroutine.
type Filter struct {
	deviceIDHeader string
	emitter        *Emitter
}

// NewFilter returns a proxy filter that drives the emitter from the
// request's device-id header. headerName is typically common.DeviceIdHeader
// so the filter package itself doesn't depend on the http-proxy-lantern
// common package.
func NewFilter(headerName string, emitter *Emitter) *Filter {
	return &Filter{deviceIDHeader: headerName, emitter: emitter}
}

// Apply implements filters.Filter. Forwards unconditionally (the
// emitter is a side-effect; failures are non-fatal).
func (f *Filter) Apply(cs *filters.ConnectionState, req *http.Request, next filters.Next) (*http.Response, *filters.ConnectionState, error) {
	if f.emitter != nil && f.emitter.Enabled() {
		f.emitter.EmitIfFirstSeen(req.Context(), req.Header.Get(f.deviceIDHeader))
	}
	return next(cs, req)
}
