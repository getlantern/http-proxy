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
	addrs, err := net.LookupHost("ats1.member.vip.ne1.yahoo.com")
	if assert.NoError(t, err, "Should have been able to resolve yahoo.com") {
		normalized := am.(*analyticsMiddleware).normalizeSite(addrs[0])
		assert.Len(t, normalized, 3, "Should have gotten two sites")
		assert.Equal(t, "ats1.member.vip.ne1.yahoo.com", normalized[1])
		assert.Equal(t, "/generated/yahoo.com", normalized[2])
	}
}
