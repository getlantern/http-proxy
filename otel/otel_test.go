package otel

import (
	"testing"

	"github.com/stretchr/testify/assert"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

// TestDeltaTemporalityCoversHistograms guards the exporter's temporality
// selection.
//
// InstrumentKindHistogram is the one that matters: it was cumulative while
// counters were already delta, and it is the kind proxy.session.goodput uses.
// Delta is required for the ops collector to strip
// route.id/instance.id/host.name from that metric — aggregating an identifier
// away merges the matching series, and merging cumulative streams interleaves
// their resets, corrupting rate() with no error. lantern-cloud also reads the
// paired .count stream with timeAggregation=sum, which is only correct for
// delta (increase vs sum on the wrong temporality is a ~500x error).
func TestDeltaTemporalityCoversHistograms(t *testing.T) {
	kinds := []sdkmetric.InstrumentKind{
		sdkmetric.InstrumentKindCounter,
		sdkmetric.InstrumentKindUpDownCounter,
		sdkmetric.InstrumentKindHistogram,
		sdkmetric.InstrumentKindGauge,
		sdkmetric.InstrumentKindObservableCounter,
		sdkmetric.InstrumentKindObservableUpDownCounter,
		sdkmetric.InstrumentKindObservableGauge,
	}
	for _, k := range kinds {
		assert.Equal(t, metricdata.DeltaTemporality, deltaTemporality(k),
			"instrument kind %v must export delta", k)
	}
}
