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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newGoodputInstrument wires a manual-reader meter provider into the global
// otel state and returns a defaultInstrument plus the reader to collect from.
func newGoodputInstrument(t *testing.T) (*defaultInstrument, *sdkmetric.ManualReader) {
	t.Helper()
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	sdkotel.SetMeterProvider(provider)

	ins, err := NewDefault(geo.NoLookup{}, &mockISPLookup{}, "test-proxy")
	require.NoError(t, err)
	return ins, reader
}

// TestSessionGoodput verifies the per-session download goodput histogram is
// recorded once for a session that moved >= goodputMinBytes, with the value
// ~= received bytes / connection seconds and a receive direction tag.
func TestSessionGoodput(t *testing.T) {
	ins, reader := newGoodputInstrument(t)

	const recv = 1_100_000 // above the 1MB goodput threshold
	ins.SessionGoodput(context.Background(), recv, time.Second, net.ParseIP("1.2.3.4"))

	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(context.Background(), &rm))

	count, sum, found := histogramCountSum(rm, "proxy.session.goodput")
	require.True(t, found, "goodput histogram should be emitted for a >=1MB session")
	assert.Equal(t, uint64(1), count, "exactly one goodput sample")
	// 1s open duration → goodput ~= received bytes per second.
	assert.InDelta(t, float64(recv), sum, float64(recv)*0.01)

	attrs := extractHistogramAttrs(rm, "proxy.session.goodput")
	assert.Equal(t, "receive", attrs["network.io.direction"])
}

// TestSessionGoodputBelowThreshold verifies a sub-threshold session records no
// goodput sample.
func TestSessionGoodputBelowThreshold(t *testing.T) {
	ins, reader := newGoodputInstrument(t)

	ins.SessionGoodput(context.Background(), 42, time.Second, net.ParseIP("1.2.3.4"))

	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(context.Background(), &rm))

	_, _, found := histogramCountSum(rm, "proxy.session.goodput")
	assert.False(t, found, "no goodput sample below the byte threshold")
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
