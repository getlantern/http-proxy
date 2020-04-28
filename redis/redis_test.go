package redis

import (
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/getlantern/measured"
	"github.com/getlantern/testredis"
	"github.com/stretchr/testify/assert"
	rclient "gopkg.in/redis.v5"

	"github.com/getlantern/http-proxy-lantern/usage"
)

func TestReportPeriodically(t *testing.T) {
	tr, err := testredis.Open()
	assert.NoError(t, err)
	defer tr.Close()
	rc := tr.Client()
	deviceID := "device12"
	clientIP := "1.1.1.1"
	fetcher := NewDeviceFetcher(rc)
	statsCh := make(chan *statsAndContext, 10000)
	newStats := func() {
		statsCh <- &statsAndContext{map[string]interface{}{"deviceid": deviceID, "client_ip": clientIP}, &measured.Stats{RecvTotal: 2, SentTotal: 1}}
	}
	lookup := &fakeLookup{}
	go reportPeriodically(lookup, rc, time.Millisecond, statsCh)

	fetcher.RequestNewDeviceUsage(deviceID)
	time.Sleep(10 * time.Millisecond)
	localCopy := usage.Get(deviceID)
	assert.Equal(t, "", localCopy.CountryCode)
	assert.EqualValues(t, 0, localCopy.Bytes)
	newStats()
	time.Sleep(10 * time.Millisecond)
	result := rc.HGetAll("_client:" + deviceID).Val()
	assert.Equal(t, "2", result["bytesIn"])
	assert.Equal(t, "1", result["bytesOut"])
	assert.Equal(t, "", result["countryCode"])
	assert.True(t, rc.TTL("_client:"+deviceID).Val() > 0, "should have set TTL to the key")
	localCopy = usage.Get(deviceID)
	assert.Equal(t, "", localCopy.CountryCode)
	assert.EqualValues(t, 3, localCopy.Bytes)

	lookup.countryCode = "ir"
	newStats()
	time.Sleep(10 * time.Millisecond)
	result = rc.HGetAll("_client:" + deviceID).Val()
	assert.Equal(t, "4", result["bytesIn"])
	assert.Equal(t, "2", result["bytesOut"])
	assert.Equal(t, "ir", result["countryCode"])
	localCopy = usage.Get(deviceID)
	assert.Equal(t, "ir", localCopy.CountryCode)
	assert.EqualValues(t, 6, localCopy.Bytes)

	lookup.countryCode = ""
	newStats()
	time.Sleep(10 * time.Millisecond)
	result = rc.HGetAll("_client:" + deviceID).Val()
	assert.Equal(t, "ir", result["countryCode"], "country code should have been remembered once set")

}

type fakeLookup struct{ countryCode string }

func (l *fakeLookup) CountryCode(ip net.IP) string {
	return l.countryCode
}
