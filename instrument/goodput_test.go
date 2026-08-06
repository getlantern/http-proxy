package instrument

import (
	"context"
	"net"
	"testing"
	"time"

	sdkotel "go.opentelemetry.io/otel"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"github.com/getlantern/geo"
	"github.com/getlantern/http-proxy-lantern/v2/instrument/otelinstrument"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newGoodputInstrument wires a manual-reader meter provider into the global
// otel state and returns a defaultInstrument plus the reader to collect from.
func newGoodputInstrument(t *testing.T) (*defaultInstrument, *sdkmetric.ManualReader) {
	t.Helper()
	// Restore the global meter provider after the test so the manual-reader
	// provider doesn't leak into other tests in the process.
	prev := sdkotel.GetMeterProvider()
	t.Cleanup(func() { sdkotel.SetMeterProvider(prev) })

	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	sdkotel.SetMeterProvider(provider)

	// Rebind otelinstrument's instruments to this provider's reader. Initialize()
	// is sync.Once-guarded, so without this reset only the first goodput test to
	// run in the process would observe recorded metrics (the rest would record
	// into the first test's now-discarded reader).
	require.NoError(t, otelinstrument.ResetForTest())

	ins, err := NewDefault(geo.NoLookup{}, &mockISPLookup{}, "test-proxy", "test-track")
	require.NoError(t, err)
	return ins, reader
}

// TestSessionGoodput verifies the per-session goodput histogram is recorded
// once for a session that moved data, with the value ~= received bytes /
// connection seconds and a receive direction tag.
func TestSessionGoodput(t *testing.T) {
	ins, reader := newGoodputInstrument(t)

	const recv = 1_100_000
	ins.SessionGoodput(context.Background(), recv, time.Second, net.ParseIP("1.2.3.4"))

	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(context.Background(), &rm))

	count, sum, found := histogramCountSum(rm, "proxy.session.goodput")
	require.True(t, found, "goodput histogram should be emitted for a session that moved data")
	assert.Equal(t, uint64(1), count, "exactly one goodput sample")
	// 1s open duration → goodput ~= received bytes per second.
	assert.InDelta(t, float64(recv), sum, float64(recv)*0.01)

	attrs := extractHistogramAttrs(rm, "proxy.session.goodput")
	assert.Equal(t, "receive", attrs["network.io.direction"])
	// track must be a point attribute keyed "track" so the bandit evaluator can
	// slice goodput per (track, country); it queries this label, not the
	// "proxy.track" resource attribute.
	assert.Equal(t, "test-track", attrs["track"], "goodput sample should carry the track point attribute")
	// The country point attribute must always be present (empty here, since the
	// test uses geo.NoLookup) so the metric stays sliceable by country.
	_, hasCountry := attrs["geo.country.iso_code"]
	assert.True(t, hasCountry, "goodput sample should carry the geo.country.iso_code attribute")
}

// TestSessionGoodputSmallSession verifies that a small-but-real session (well
// under the old 1 MB floor) now records a goodput sample. This is the core of
// the emission fix: such sessions dominate real traffic in censored markets, and
// the old floor erased them, false-starving healthy challengers.
func TestSessionGoodputSmallSession(t *testing.T) {
	ins, reader := newGoodputInstrument(t)

	// 20 KB received over 2s — comfortably below the old 1 MB floor.
	const recv = 20 * 1024
	ins.SessionGoodput(context.Background(), recv, 2*time.Second, net.ParseIP("1.2.3.4"))

	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(context.Background(), &rm))

	count, sum, found := histogramCountSum(rm, "proxy.session.goodput")
	require.True(t, found, "goodput histogram should now be emitted for a small session")
	assert.Equal(t, uint64(1), count, "exactly one goodput sample")
	// 20 KB over 2s → ~10 KB/s.
	assert.InDelta(t, float64(recv)/2.0, sum, float64(recv)*0.01)
}

// TestSessionGoodputZeroBytes verifies a session that moved no received bytes
// records nothing (guards against dividing meaningfully on empty connections).
func TestSessionGoodputZeroBytes(t *testing.T) {
	ins, reader := newGoodputInstrument(t)

	ins.SessionGoodput(context.Background(), 0, time.Second, net.ParseIP("1.2.3.4"))

	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(context.Background(), &rm))

	_, _, found := histogramCountSum(rm, "proxy.session.goodput")
	assert.False(t, found, "no goodput sample when no bytes were received")
}

// TestSessionGoodputZeroDuration verifies a non-positive duration records no
// sample (guards against divide-by-zero).
func TestSessionGoodputZeroDuration(t *testing.T) {
	ins, reader := newGoodputInstrument(t)

	ins.SessionGoodput(context.Background(), 2_000_000, 0, net.ParseIP("1.2.3.4"))

	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(context.Background(), &rm))

	_, _, found := histogramCountSum(rm, "proxy.session.goodput")
	assert.False(t, found, "no goodput sample for a zero-duration session")
}

// TestSessionGoodputBucketLayout verifies the emitted histogram carries the
// explicit log-scale bucket boundaries instead of the SDK defaults, whose
// 10,000 B/s top boundary censored everything faster than 10 kB/s into the
// final bucket. A ~101 kB/s sample must resolve into the (100_000, 300_000]
// bucket, which only exists in the explicit layout.
func TestSessionGoodputBucketLayout(t *testing.T) {
	ins, reader := newGoodputInstrument(t)

	// 202 KB over 2s → ~101 kB/s, ten times the SDK default's top boundary.
	ins.SessionGoodput(context.Background(), 202_000, 2*time.Second, net.ParseIP("1.2.3.4"))

	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(context.Background(), &rm))

	dp, found := histogramDataPoint(rm, "proxy.session.goodput")
	require.True(t, found, "goodput histogram should be emitted")
	assert.Equal(t, otelinstrument.GoodputBucketBoundaries, dp.Bounds,
		"histogram must carry the explicit log-scale boundaries")
	require.Len(t, dp.BucketCounts, len(otelinstrument.GoodputBucketBoundaries)+1)
	// Boundaries are upper-inclusive: 101_000 lands in (100_000, 300_000],
	// i.e. bucket index 11 — not the overflow bucket.
	assert.Equal(t, uint64(1), dp.BucketCounts[11], "~101 kB/s sample should land in the (100k, 300k] bucket")
}

func histogramDataPoint(rm metricdata.ResourceMetrics, name string) (metricdata.HistogramDataPoint[float64], bool) {
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != name {
				continue
			}
			if d, ok := m.Data.(metricdata.Histogram[float64]); ok && len(d.DataPoints) > 0 {
				return d.DataPoints[0], true
			}
		}
	}
	return metricdata.HistogramDataPoint[float64]{}, false
}

func histogramCountSum(rm metricdata.ResourceMetrics, name string) (uint64, float64, bool) {
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != name {
				continue
			}
			if d, ok := m.Data.(metricdata.Histogram[float64]); ok && len(d.DataPoints) > 0 {
				return d.DataPoints[0].Count, d.DataPoints[0].Sum, true
			}
		}
	}
	return 0, 0, false
}

func extractHistogramAttrs(rm metricdata.ResourceMetrics, name string) map[string]string {
	result := make(map[string]string)
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != name {
				continue
			}
			if d, ok := m.Data.(metricdata.Histogram[float64]); ok && len(d.DataPoints) > 0 {
				for _, kv := range d.DataPoints[0].Attributes.ToSlice() {
					result[string(kv.Key)] = kv.Value.Emit()
				}
			}
		}
	}
	return result
}
