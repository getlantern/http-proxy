package datacap

import (
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/getlantern/http-proxy-lantern/v2/common"
	"github.com/getlantern/measured"
)

const testDefaultRate = int64(640000)

type fixedCountry string

func (c fixedCountry) CountryCode(net.IP) string { return string(c) }

// fakeSidecar accumulates reported deltas the way the real sidecar does and
// throttles once the cap is exceeded.
type fakeSidecar struct {
	*httptest.Server

	mu       sync.Mutex
	reports  []Report
	total    int64
	capLimit int64
	fail     bool
	delay    time.Duration
}

func newFakeSidecar(capLimit int64) *fakeSidecar {
	s := &fakeSidecar{capLimit: capLimit}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var report Report
		if err := json.NewDecoder(r.Body).Decode(&report); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		s.mu.Lock()
		if s.fail {
			s.mu.Unlock()
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		delay := s.delay
		if delay > 0 && report.DeviceID == "slowpoke" {
			s.mu.Unlock()
			time.Sleep(delay)
			s.mu.Lock()
		}
		s.reports = append(s.reports, report)
		s.total += report.BytesUsed
		status := Status{
			Throttle:   s.capLimit > 0 && s.total >= s.capLimit,
			CapLimit:   s.capLimit,
			ExpiryTime: time.Now().Add(6 * time.Hour).Unix(),
			BytesUsed:  s.total,
		}
		s.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(status)
	}))
	return s
}

func (s *fakeSidecar) snapshot() (reports []Report, total int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Report(nil), s.reports...), s.total
}

func (s *fakeSidecar) setFail(fail bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.fail = fail
}

func newTestTracker(t *testing.T, sidecar *fakeSidecar) *Tracker {
	t.Helper()
	return NewTracker(TrackerOpts{
		Client:         NewClient(sidecar.URL, time.Second),
		CountryLookup:  fixedCountry("ES"),
		DefaultRate:    testDefaultRate,
		ReportInterval: 10 * time.Millisecond,
	})
}

func report(t *Tracker, deviceID string, bytes int) {
	t.Reporter()(map[string]interface{}{
		common.DeviceID: deviceID,
		common.ClientIP: "1.2.3.4",
		common.Platform: "android",
	}, &measured.Stats{}, &measured.Stats{RecvTotal: bytes}, false)
}

// The whole point of the shared limiter: a device that crosses its cap is
// slowed on the limiter its already-open connections hold, not on a fresh one
// handed to the next connection.
func TestThrottleAppliesToTheLimiterAlreadyHandedOut(t *testing.T) {
	sidecar := newFakeSidecar(1000)
	defer sidecar.Close()
	tracker := newTestTracker(t, sidecar)

	limiter := tracker.Limiter("device1", false)
	require.Equal(t, testDefaultRate, limiter.GetRateWrite(), "should start at the default rate")

	report(tracker, "device1", 400)
	assert.Eventually(t, func() bool {
		u, ok := tracker.Usage("device1")
		return ok && u.BytesUsed == 400 && !u.Throttled
	}, time.Second, 5*time.Millisecond, "usage should be reported below the cap without throttling")
	assert.Equal(t, testDefaultRate, limiter.GetRateWrite(), "should not be throttled below the cap")

	report(tracker, "device1", 700)
	assert.Eventually(t, func() bool {
		return limiter.GetRateWrite() == ThrottledWriteRate
	}, time.Second, 5*time.Millisecond, "crossing the cap should re-rate the limiter that was already handed out")
	assert.Equal(t, testDefaultRate, limiter.GetRateRead(), "uploads stay at the default rate when capped")
	assert.Same(t, limiter, tracker.Limiter("device1", false), "the device's limiter is shared, not rebuilt")
}

// Bytes to cap-excluded domains still count, they are just never slowed.
func TestUnthrottledLimiterIsNeverCapped(t *testing.T) {
	sidecar := newFakeSidecar(100)
	defer sidecar.Close()
	tracker := newTestTracker(t, sidecar)

	unthrottled := tracker.Limiter("device1", true)
	report(tracker, "device1", 500)
	assert.Eventually(t, func() bool {
		u, ok := tracker.Usage("device1")
		return ok && u.Throttled
	}, time.Second, 5*time.Millisecond)

	assert.Equal(t, testDefaultRate, unthrottled.GetRateWrite())
	assert.Equal(t, ThrottledWriteRate, tracker.Limiter("device1", false).GetRateWrite())
}

func TestDeltasAreAggregatedPerDevice(t *testing.T) {
	sidecar := newFakeSidecar(0)
	defer sidecar.Close()
	tracker := newTestTracker(t, sidecar)

	for i := 0; i < 5; i++ {
		report(tracker, "device1", 10)
		report(tracker, "device2", 3)
	}

	assert.Eventually(t, func() bool {
		_, total := sidecar.snapshot()
		return total == 65
	}, time.Second, 5*time.Millisecond)

	reports, _ := sidecar.snapshot()
	byDevice := map[string]int64{}
	for _, r := range reports {
		byDevice[r.DeviceID] += r.BytesUsed
		assert.Equal(t, "ES", r.CountryCode, "country comes from the client IP lookup")
		assert.Equal(t, "android", r.Platform)
	}
	assert.Equal(t, int64(50), byDevice["device1"])
	assert.Equal(t, int64(15), byDevice["device2"])
	assert.Less(t, len(reports), 10, "deltas should be batched, not posted one per call")
}

// A sidecar that is down must not silently eat usage.
func TestFailedReportsAreRetried(t *testing.T) {
	sidecar := newFakeSidecar(0)
	defer sidecar.Close()
	sidecar.setFail(true)
	tracker := newTestTracker(t, sidecar)

	report(tracker, "device1", 250)
	time.Sleep(50 * time.Millisecond)
	_, total := sidecar.snapshot()
	require.Zero(t, total, "nothing should be recorded while the sidecar is failing")

	sidecar.setFail(false)
	assert.Eventually(t, func() bool {
		_, total := sidecar.snapshot()
		return total == 250
	}, time.Second, 5*time.Millisecond, "the delta should be retried once the sidecar recovers")
}

func TestUsageIsUnknownUntilTheSidecarAnswers(t *testing.T) {
	sidecar := newFakeSidecar(1000)
	defer sidecar.Close()
	tracker := newTestTracker(t, sidecar)

	_, ok := tracker.Usage("device1")
	assert.False(t, ok)
	assert.Equal(t, testDefaultRate, tracker.Limiter("device1", false).GetRateWrite(),
		"an unknown device runs at the default rate, not throttled")
}

// The stats buffer only fills when the reporting loop is stalled on the
// sidecar, which is exactly when a device is most likely to be running past its
// cap. Deltas must survive that rather than be dropped.
func TestOverflowingDeltasAreNotLost(t *testing.T) {
	sidecar := newFakeSidecar(0)
	defer sidecar.Close()
	tracker := newTestTracker(t, sidecar)

	// Far more deltas than the buffer holds, submitted without letting the
	// reporting loop drain in between.
	const reports = statsBufferSize * 2
	for i := 0; i < reports; i++ {
		report(tracker, "device1", 1)
	}

	assert.Eventually(t, func() bool {
		_, total := sidecar.snapshot()
		return total == int64(reports)
	}, 5*time.Second, 10*time.Millisecond, "every delta should reach the sidecar")
}

func TestDeltasWithoutADeviceIDAreIgnored(t *testing.T) {
	sidecar := newFakeSidecar(0)
	defer sidecar.Close()
	tracker := newTestTracker(t, sidecar)

	reporter := tracker.Reporter()
	for _, ctx := range []map[string]interface{}{
		{common.ClientIP: "1.2.3.4"},                      // absent
		{common.DeviceID: "", common.ClientIP: "1.2.3.4"}, // present but empty
		{common.DeviceID: 42, common.ClientIP: "1.2.3.4"}, // wrong type
	} {
		reporter(ctx, &measured.Stats{}, &measured.Stats{RecvTotal: 100}, false)
	}

	time.Sleep(100 * time.Millisecond)
	reports, total := sidecar.snapshot()
	assert.Zero(t, total)
	assert.Empty(t, reports)
}

// One unresponsive device must not hold up the throttle verdict for every other
// device in the batch.
func TestASlowDeviceDoesNotDelayTheBatch(t *testing.T) {
	sidecar := newFakeSidecar(1000)
	defer sidecar.Close()
	sidecar.mu.Lock()
	sidecar.delay = 750 * time.Millisecond
	sidecar.mu.Unlock()

	tracker := newTestTracker(t, sidecar)
	limiter := tracker.Limiter("device1", false)

	report(tracker, "slowpoke", 5)
	report(tracker, "device1", 2000)

	assert.Eventually(t, func() bool {
		return limiter.GetRateWrite() == ThrottledWriteRate
	}, 500*time.Millisecond, 5*time.Millisecond,
		"device1 should be throttled well before the slow device's report returns")
}
