package instrument

import (
	"context"
	"net"
	"testing"

	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"github.com/getlantern/semconv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestProxiedBytesProxyIODeviceID verifies proxy.io carries the client's
// device ID in both directions, so per-protocol user counts can come from
// proxy.io alone.
func TestProxiedBytesProxyIODeviceID(t *testing.T) {
	ins, reader := newGoodputInstrument(t)

	ins.ProxiedBytes(context.Background(), 100, 200,
		"android", "14", "9.0.1", "9.0.1", "lantern", "en_US", "", "",
		net.ParseIP("1.2.3.4"), "dev-123", "", "arm64")

	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(context.Background(), &rm))

	byDirection := map[string]int64{}
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != semconv.ProxyMetricIO {
				continue
			}
			sum, ok := m.Data.(metricdata.Sum[int64])
			require.True(t, ok, "proxy.io should be an int64 sum")
			for _, dp := range sum.DataPoints {
				deviceID, found := dp.Attributes.Value(semconv.ClientDeviceIDKey)
				require.True(t, found, "proxy.io datapoint should carry client.device_id")
				assert.Equal(t, "dev-123", deviceID.AsString())
				direction, _ := dp.Attributes.Value(semconv.NetworkIODirectionKey)
				byDirection[direction.AsString()] += dp.Value
			}
		}
	}
	assert.Equal(t, map[string]int64{"transmit": 100, "receive": 200}, byDirection)
}
