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

	// statsBufferSize bounds the queue of unaggregated deltas.
	statsBufferSize = 10000

	// flushConcurrency bounds the reports in flight during one cycle, so a
	// proxy with many active devices does not open an unbounded number of
	// connections to the sidecar.
	flushConcurrency = 16
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
	// AsOf is when the sidecar reported these numbers.
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

	usage     Usage
	haveUsage bool

	// pendingBytes is the delta not yet accepted by the sidecar.
	pendingBytes int64
	// countryCode, platform are the most recent values seen for this device;
	// the sidecar keys the cap limit off them.
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
	throttledRate  int64
	reportInterval time.Duration

	mx      sync.RWMutex
	devices map[string]*device

	statsCh chan *statsAndContext
}

type statsAndContext struct {
	ctx   map[string]interface{}
	stats *measured.Stats
}

// TrackerOpts configures a Tracker.
type TrackerOpts struct {
	Client         *Client
	CountryLookup  geo.CountryLookup
	DefaultRate    int64
	ThrottledRate  int64
	ReportInterval time.Duration
}

// NewTracker starts a Tracker and its reporting loop.
func NewTracker(opts TrackerOpts) *Tracker {
	if opts.ReportInterval <= 0 {
		opts.ReportInterval = DefaultReportInterval
	}
	if opts.ThrottledRate <= 0 {
		opts.ThrottledRate = ThrottledWriteRate
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
		throttledRate:  opts.ThrottledRate,
		reportInterval: opts.ReportInterval,
		devices:        make(map[string]*device),
		statsCh:        make(chan *statsAndContext, statsBufferSize),
	}
	go t.reportPeriodically()
	return t
}

// Reporter returns the callback the measured listener feeds connection deltas
// into.
func (t *Tracker) Reporter() listeners.MeasuredReportFN {
	return func(ctx map[string]interface{}, stats *measured.Stats, deltaStats *measured.Stats, final bool) {
		if deltaStats.SentTotal == 0 && deltaStats.RecvTotal == 0 {
			return
		}
		if deviceID, _ := ctx[common.DeviceID].(string); deviceID == "" {
			// Nothing to account against, so don't spend buffer capacity on it.
			return
		}
		sac := &statsAndContext{ctx, deltaStats}
		select {
		case t.statsCh <- sac:
		default:
			// The buffer only fills if the reporting loop is stalled, which is
			// exactly when a device is most likely to be running past its cap.
			// Fold the delta in directly rather than lose it: accumulate takes
			// the tracker lock, which is never held across a sidecar call, so
			// this cannot block on the network.
			t.accumulate(sac)
		}
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

// Usage returns the last cap state the sidecar reported for deviceID. ok is
// false until the first report for that device has been answered.
func (t *Tracker) Usage(deviceID string) (Usage, bool) {
	t.mx.RLock()
	defer t.mx.RUnlock()
	d, exists := t.devices[deviceID]
	if !exists || !d.haveUsage {
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
	for {
		select {
		case sac := <-t.statsCh:
			t.accumulate(sac)
		case <-ticker.C:
			// Bound the whole cycle. Reports run concurrently, but a wedged
			// sidecar would still hold this goroutine for a full HTTP timeout,
			// and for that entire window no throttle verdict is applied and
			// nothing drains statsCh. Deadlined reports are restored and
			// retried on the next tick like any other failure.
			ctx, cancel := context.WithTimeout(context.Background(), t.flushTimeout())
			t.flush(ctx)
			cancel()
		}
	}
}

// flushTimeout bounds one reporting cycle. It never drops below the per-request
// timeout, so a healthy-but-slow sidecar is not cut off by an aggressive
// reportInterval.
func (t *Tracker) flushTimeout() time.Duration {
	if d := 2 * t.reportInterval; d > DefaultHTTPTimeout {
		return d
	}
	return DefaultHTTPTimeout
}

func (t *Tracker) accumulate(sac *statsAndContext) {
	deviceID, _ := sac.ctx[common.DeviceID].(string)
	if deviceID == "" {
		return
	}
	d := t.deviceFor(deviceID)

	countryCode := ""
	if clientIP, ok := sac.ctx[common.ClientIP].(string); ok {
		countryCode = t.countryLookup.CountryCode(net.ParseIP(clientIP))
	}
	platform, _ := sac.ctx[common.Platform].(string)

	t.mx.Lock()
	d.pendingBytes += int64(sac.stats.SentTotal) + int64(sac.stats.RecvTotal)
	if countryCode != "" {
		d.countryCode = countryCode
	}
	if platform != "" {
		d.platform = platform
	}
	d.lastSeen = time.Now()
	t.mx.Unlock()
}

// pendingReport is one device's delta, detached from the tracker so the HTTP
// round-trips happen without holding the lock.
type pendingReport struct {
	deviceID string
	device   *device
	report   Report
}

func (t *Tracker) flush(ctx context.Context) {
	t.mx.Lock()
	var reports []pendingReport
	for deviceID, d := range t.devices {
		if d.pendingBytes == 0 {
			continue
		}
		reports = append(reports, pendingReport{
			deviceID: deviceID,
			device:   d,
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

	// Report concurrently: serially, one slow device delays the throttle
	// verdict for every device behind it in the batch, and the cycle's duration
	// grows with the number of active devices.
	var (
		failed  atomic.Int64
		lastErr atomic.Value
		wg      sync.WaitGroup
	)
	sem := make(chan struct{}, flushConcurrency)
	for _, pr := range reports {
		wg.Add(1)
		sem <- struct{}{}
		go func(pr pendingReport) {
			defer wg.Done()
			defer func() { <-sem }()

			status, err := t.client.ReportUsage(ctx, &pr.report)
			if err != nil {
				failed.Add(1)
				lastErr.Store(err)
				t.restore(pr)
				return
			}
			t.apply(pr.device, status)
		}(pr)
	}
	wg.Wait()

	if n := failed.Load(); n > 0 {
		// A wedged sidecar fails every device in the batch, so log once per
		// cycle rather than once per device.
		err, _ := lastErr.Load().(error)
		log.Errorf("Unable to report usage for %d of %d devices, will retry: %v", n, len(reports), err)
	}

	t.evictIdle()
}

// restore puts a failed report's bytes back so the next cycle retries them.
func (t *Tracker) restore(pr pendingReport) {
	t.mx.Lock()
	defer t.mx.Unlock()
	pr.device.pendingBytes += pr.report.BytesUsed
}

// apply records the sidecar's answer and re-rates the device's shared limiter.
// Only writes back to the client are throttled: a capped device can keep
// uploading at the default rate.
func (t *Tracker) apply(d *device, status *Status) {
	now := time.Now()
	usage := Usage{
		BytesUsed: status.BytesUsed,
		CapLimit:  status.CapLimit,
		AsOf:      now,
		Throttled: status.Throttle,
	}
	if status.ExpiryTime > 0 {
		usage.Expiry = time.Unix(status.ExpiryTime, 0)
	}

	t.mx.Lock()
	d.usage = usage
	d.haveUsage = true
	t.mx.Unlock()

	if status.Throttle {
		d.limiter.SetRates(t.defaultRate, t.throttledRate)
	} else {
		d.limiter.SetRates(t.defaultRate, t.defaultRate)
	}
}

// idleDeviceTTL is how long a device with nothing pending is kept around. It
// outlives the measured reporting interval by a wide margin so a device with a
// quiet connection does not lose its throttle state and get a free window at
// full speed.
const idleDeviceTTL = 30 * time.Minute

func (t *Tracker) evictIdle() {
	cutoff := time.Now().Add(-idleDeviceTTL)
	t.mx.Lock()
	defer t.mx.Unlock()
	for deviceID, d := range t.devices {
		if d.lastSeen.Before(cutoff) && d.pendingBytes == 0 {
			delete(t.devices, deviceID)
		}
	}
}
