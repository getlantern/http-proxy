package datacap

import (
	"context"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/getlantern/geo"
	"github.com/getlantern/golog"
	"github.com/getlantern/measured"

	"github.com/getlantern/http-proxy-lantern/v2/common"
	"github.com/getlantern/http-proxy-lantern/v2/listeners"
)

var log = golog.LoggerFor("datacap")

const (
	// DefaultReportInterval is how often accumulated deltas are flushed to the
	// sidecar. The measured pipeline hands us per-connection deltas on its own
	// (slower) cadence; flushing more often than that just shortens the gap
	// between a delta arriving and the throttle verdict coming back.
	DefaultReportInterval = 10 * time.Second

	// ThrottledWriteRate is the download rate a capped device is held to, in
	// bytes per second. Matches lantern-box (tracker/datacap/conn.go) so a
	// device sees the same speed whichever proxy flavor it lands on.
	ThrottledWriteRate int64 = 16 * 1024 // 128 Kb/s

	// flushConcurrency bounds the reports in flight during one cycle, so a
	// proxy with many active devices does not open an unbounded number of
	// connections to the sidecar.
	flushConcurrency = 16

	// idleDeviceTTL is how long a device with nothing pending is kept around.
	// Evicting a device severs its open connections from re-rating (they keep
	// the limiter pointer they were attached with, while a re-appearing device
	// gets a fresh entry), so this must comfortably exceed the proxy's
	// idle-close timeout (`idleclose`, ~70-90s): a connection idle longer than
	// that is closed by the proxy itself, and one that is still alive is moving
	// bytes, whose deltas refresh lastSeen and block eviction. At 30 minutes
	// the margin over idleclose plus the measured reporting interval is >10x.
	idleDeviceTTL = 30 * time.Minute
)

// Usage is a device's cap state as of the last sidecar response.
type Usage struct {
	// BytesUsed is the total consumed in the current allotment period.
	BytesUsed int64
	// CapLimit is the allotment in bytes. Zero means this device is uncapped
	// (no cap entry for its country/platform).
	CapLimit int64
	// Expiry is when the current allotment resets.
	Expiry time.Time
	// AsOf is when the sidecar reported these numbers. A zero AsOf means the
	// sidecar has not answered for this device yet.
	AsOf time.Time
	// Throttled is the sidecar's verdict at AsOf.
	Throttled bool
}

// device holds the per-device state shared between the reporting loop and the
// request filter.
type device struct {
	// limiter is attached to every connection of this device and re-rated in
	// place when the throttle verdict changes, so a transfer that is already
	// running slows down at the moment the cap is crossed instead of at the
	// next CONNECT.
	limiter *listeners.RateLimiter
	// unthrottledLimiter serves requests to domains excluded from the cap. It
	// stays at the default rate for the device's lifetime; those requests are
	// still counted, they are just never slowed to the capped rate.
	unthrottledLimiter *listeners.RateLimiter

	usage Usage

	// pendingBytes is the delta not yet accepted by the sidecar.
	pendingBytes int64
	// countryCode, platform key the sidecar's cap-limit lookup. countryCode is
	// sticky — set once from the first delta that resolves one — matching the
	// reporting-Redis behavior this replaces.
	countryCode string
	platform    string
	// lastSeen gates eviction of idle devices.
	lastSeen time.Time
}

// Tracker aggregates per-device byte usage, reports it to the sidecar, and owns
// the rate limiters the verdict is enforced through.
type Tracker struct {
	client        *Client
	countryLookup geo.CountryLookup
	// defaultRate is the ceiling every non-pro device is held to regardless of
	// its cap state, to keep bandwidth hogs from monopolizing a proxy.
	defaultRate    int64
	reportInterval time.Duration

	mx      sync.RWMutex
	devices map[string]*device
}

// TrackerOpts configures a Tracker.
type TrackerOpts struct {
	Client         *Client
	CountryLookup  geo.CountryLookup
	DefaultRate    int64
	ReportInterval time.Duration
}

// NewTracker starts a Tracker and its reporting loop.
func NewTracker(opts TrackerOpts) *Tracker {
	if opts.ReportInterval <= 0 {
		opts.ReportInterval = DefaultReportInterval
	}
	if opts.CountryLookup == nil {
		// Reports still carry the device and its platform; only the
		// country-specific cap limit is lost. That beats panicking in the
		// reporting loop.
		opts.CountryLookup = geo.NoLookup{}
	}
	t := &Tracker{
		client:         opts.Client,
		countryLookup:  opts.CountryLookup,
		defaultRate:    opts.DefaultRate,
		reportInterval: opts.ReportInterval,
		devices:        make(map[string]*device),
	}
	go t.reportPeriodically()
	return t
}

// Reporter returns the callback the measured listener feeds connection deltas
// into. Deltas are folded into per-device state synchronously: the tracker lock
// is never held across a sidecar call, so this cannot block on the network.
func (t *Tracker) Reporter() listeners.MeasuredReportFN {
	return func(ctx map[string]interface{}, stats *measured.Stats, deltaStats *measured.Stats, final bool) {
		bytes := int64(deltaStats.SentTotal) + int64(deltaStats.RecvTotal)
		if bytes == 0 {
			return
		}
		deviceID, _ := ctx[common.DeviceID].(string)
		if deviceID == "" {
			// Nothing to account against.
			return
		}
		clientIP, _ := ctx[common.ClientIP].(string)
		platform, _ := ctx[common.Platform].(string)
		t.accumulate(deviceID, clientIP, platform, bytes)
	}
}

// Limiter returns the shared limiter to attach to a connection from deviceID.
// unthrottled selects the limiter used for requests to domains excluded from
// the cap, which is never re-rated to the capped speed.
func (t *Tracker) Limiter(deviceID string, unthrottled bool) *listeners.RateLimiter {
	d := t.deviceFor(deviceID)
	if unthrottled {
		return d.unthrottledLimiter
	}
	return d.limiter
}

// LimiterAndUsage returns the device's shared limiter together with the last
// cap state the sidecar reported for it, in one lookup. ok is false until the
// first report for that device has been answered.
func (t *Tracker) LimiterAndUsage(deviceID string) (limiter *listeners.RateLimiter, u Usage, ok bool) {
	d := t.deviceFor(deviceID)
	t.mx.RLock()
	u = d.usage
	t.mx.RUnlock()
	return d.limiter, u, !u.AsOf.IsZero()
}

// Usage returns the last cap state the sidecar reported for deviceID. ok is
// false until the first report for that device has been answered.
func (t *Tracker) Usage(deviceID string) (Usage, bool) {
	t.mx.RLock()
	defer t.mx.RUnlock()
	d, exists := t.devices[deviceID]
	if !exists || d.usage.AsOf.IsZero() {
		return Usage{}, false
	}
	return d.usage, true
}

func (t *Tracker) deviceFor(deviceID string) *device {
	t.mx.RLock()
	d, exists := t.devices[deviceID]
	t.mx.RUnlock()
	if exists {
		return d
	}

	t.mx.Lock()
	defer t.mx.Unlock()
	if d, exists = t.devices[deviceID]; exists {
		return d
	}
	d = &device{
		limiter:            listeners.NewRateLimiter(t.defaultRate, t.defaultRate),
		unthrottledLimiter: listeners.NewRateLimiter(t.defaultRate, t.defaultRate),
		lastSeen:           time.Now(),
	}
	t.devices[deviceID] = d
	return d
}

func (t *Tracker) reportPeriodically() {
	ticker := time.NewTicker(t.reportInterval)
	defer ticker.Stop()
	for range ticker.C {
		// Bound the whole cycle. Reports run concurrently, but a wedged
		// sidecar would still hold this goroutine for a full HTTP timeout,
		// and for that entire window no throttle verdict is applied.
		// Deadlined reports are restored and retried on the next tick like
		// any other failure. The deadline never drops below the per-request
		// timeout, so a healthy-but-slow sidecar is not cut off by an
		// aggressive reportInterval.
		timeout := 2 * t.reportInterval
		if timeout < DefaultHTTPTimeout {
			timeout = DefaultHTTPTimeout
		}
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		t.flush(ctx)
		cancel()
	}
}

func (t *Tracker) accumulate(deviceID, clientIP, platform string, bytes int64) {
	d := t.deviceFor(deviceID)

	// The country lookup walks the MaxMind database, so do it at most once per
	// device — outside the lock — rather than on every delta.
	t.mx.RLock()
	needCountry := d.countryCode == ""
	t.mx.RUnlock()
	countryCode := ""
	if needCountry && clientIP != "" {
		countryCode = t.countryLookup.CountryCode(net.ParseIP(clientIP))
	}

	now := time.Now()
	t.mx.Lock()
	d.pendingBytes += bytes
	if d.countryCode == "" {
		d.countryCode = countryCode
	}
	if platform != "" {
		d.platform = platform
	}
	d.lastSeen = now
	t.mx.Unlock()
}

// pendingReport is one device's delta, detached from the tracker so the HTTP
// round-trips happen without holding the lock.
type pendingReport struct {
	device *device
	report Report
}

func (t *Tracker) flush(ctx context.Context) {
	evictBefore := time.Now().Add(-idleDeviceTTL)

	t.mx.Lock()
	reports := make([]pendingReport, 0, len(t.devices))
	for deviceID, d := range t.devices {
		if d.pendingBytes == 0 {
			// The flush already visits every device, so idle eviction rides
			// along instead of taking a second full scan under the lock.
			if d.lastSeen.Before(evictBefore) {
				delete(t.devices, deviceID)
			}
			continue
		}
		reports = append(reports, pendingReport{
			device: d,
			report: Report{
				DeviceID:    deviceID,
				CountryCode: d.countryCode,
				Platform:    d.platform,
				BytesUsed:   d.pendingBytes,
			},
		})
		d.pendingBytes = 0
	}
	t.mx.Unlock()

	if len(reports) == 0 {
		return
	}

	// Report concurrently: serially, one slow device delays the throttle
	// verdict for every device behind it in the batch, and the cycle's
	// duration grows with the number of active devices.
	statuses := make([]*Status, len(reports))
	errs := make([]error, len(reports))
	workers := flushConcurrency
	if len(reports) < workers {
		workers = len(reports)
	}
	var next atomic.Int64
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				i := int(next.Add(1)) - 1
				if i >= len(reports) {
					return
				}
				pr := reports[i]
				statuses[i], errs[i] = t.client.ReportUsage(ctx, &pr.report)
				if st := statuses[i]; st != nil {
					// Re-rate as soon as the verdict is in — SetRates is
					// lock-free, and waiting for the whole batch would let one
					// slow device delay enforcement for every other device.
					// Only writes back to the client are throttled: a capped
					// device can keep uploading at the default rate.
					if st.Throttle {
						pr.device.limiter.SetRates(t.defaultRate, ThrottledWriteRate)
					} else {
						pr.device.limiter.SetRates(t.defaultRate, t.defaultRate)
					}
				}
			}
		}()
	}
	wg.Wait()

	// The usage bookkeeping feeds the XBQ headers, which tolerate a batch's
	// worth of staleness — record the whole cycle under one lock acquisition
	// instead of one per device.
	now := time.Now()
	failed := 0
	var lastErr error
	t.mx.Lock()
	for i, pr := range reports {
		if errs[i] != nil {
			failed++
			lastErr = errs[i]
			// Put the bytes back so the next cycle retries them.
			pr.device.pendingBytes += pr.report.BytesUsed
			continue
		}
		st := statuses[i]
		u := Usage{
			BytesUsed: st.BytesUsed,
			CapLimit:  st.CapLimit,
			AsOf:      now,
			Throttled: st.Throttle,
		}
		if st.ExpiryTime > 0 {
			u.Expiry = time.Unix(st.ExpiryTime, 0)
		}
		pr.device.usage = u
	}
	t.mx.Unlock()

	if failed > 0 {
		// A wedged sidecar fails every device in the batch, so log once per
		// cycle rather than once per device.
		log.Errorf("Unable to report usage for %d of %d devices, will retry: %v", failed, len(reports), lastErr)
	}
}
