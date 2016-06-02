package analytics

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeSite(t *testing.T) {
	am := New(&Options{
		TrackingID:       "12345",
		SamplePercentage: 1,
	})
	addrs, err := net.LookupHost("edge-star-mini-shv-07-frc3.facebook.com")
	if assert.NoError(t, err, "Should have been able to resolve facebook.com") {
		normalized := am.(*analyticsMiddleware).normalizeSite(addrs[0])
		assert.Len(t, normalized, 3, "Should have gotten two sites")
		assert.Equal(t, "edge-star-mini-shv-07-frc3.facebook.com", normalized[1])
		assert.Equal(t, "/generated/facebook.com", normalized[2])
	}
}
