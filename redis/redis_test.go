package redis

import (
	"context"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/getlantern/measured"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/getlantern/http-proxy-lantern/v2/internal/testutil"
	"github.com/getlantern/http-proxy-lantern/v2/throttle"
	"github.com/getlantern/http-proxy-lantern/v2/usage"
)

func TestRedisUrl(t *testing.T) {
	cl, err := NewClient("rediss://:password@host:6379")
	assert.NoError(t, err)
	assert.NotNil(t, cl)

	cl, err = NewClient("rediss://127.0.0.1:5252")
	assert.NoError(t, err)
	assert.NotNil(t, cl)
}

func TestReportPeriodically(t *testing.T) {
	redisClient := testutil.TestRedis(t)

	deviceID := "device12"
	clientIP := "1.1.1.1"
	fetcher := NewDeviceFetcher(redisClient)
	statsCh := make(chan *statsAndContext, 10000)
	newStats := func() {
		statsCh <- &statsAndContext{map[string]interface{}{"deviceid": deviceID, "client_ip": clientIP, "app_platform": "windows", "throttled": true}, &measured.Stats{RecvTotal: 2, SentTotal: 1}}
	}
	lookup := &fakeLookup{}
	go reportPeriodically(lookup, redisClient, time.Millisecond, throttle.NewForcedConfig(5000, 500, throttle.Monthly), statsCh)

	fetcher.RequestNewDeviceUsage(deviceID)
	time.Sleep(100 * time.Millisecond)
	localCopy := usage.Get(deviceID)
	assert.Equal(t, "", localCopy.CountryCode)
	assert.EqualValues(t, 0, localCopy.Bytes)
	newStats()
	time.Sleep(300 * time.Millisecond)
	result := redisClient.HGetAll(context.Background(), "_client:"+deviceID).Val()
	assert.Equal(t, "2", result["bytesIn"])
	assert.Equal(t, "1", result["bytesOut"])
	assert.Equal(t, "", result["countryCode"])
	assert.True(t, redisClient.TTL(context.Background(), "_client:"+deviceID).Val() > 0, "should have set TTL to the key")
	localCopy = usage.Get(deviceID)
	assert.Equal(t, "", localCopy.CountryCode)
	assert.EqualValues(t, 3, localCopy.Bytes)

	lookup.countryCode = "ir"
	newStats()
	time.Sleep(10 * time.Millisecond)
	result = redisClient.HGetAll(context.Background(), "_client:"+deviceID).Val()
	assert.Equal(t, "4", result["bytesIn"])
	assert.Equal(t, "2", result["bytesOut"])
	assert.Equal(t, "ir", result["countryCode"])
	localCopy = usage.Get(deviceID)
	assert.Equal(t, "ir", localCopy.CountryCode)
	assert.EqualValues(t, 6, localCopy.Bytes)

	lookup.countryCode = ""
	newStats()
	time.Sleep(10 * time.Millisecond)
	result = redisClient.HGetAll(context.Background(), "_client:"+deviceID).Val()
	assert.Equal(t, "ir", result["countryCode"], "country code should have been remembered once set")

	uniqueDevicesForToday := redisClient.SMembers(context.Background(), "_devices:ir:"+time.Now().In(time.UTC).Format("2006-01-02")+":forced").Val()
	assert.Equal(t, []string{deviceID}, uniqueDevicesForToday)

	_deviceLastSeen := redisClient.Get(context.Background(), "_deviceLastSeen:ir:forced:"+deviceID).Val()
	deviceLastSeen, err := strconv.Atoi(_deviceLastSeen)
	require.NoError(t, err)
	_deviceFirstThrottled := redisClient.Get(context.Background(), "_deviceFirstThrottled:"+deviceID).Val()
	deviceFirstThrottled, err := strconv.Atoi(_deviceFirstThrottled)

	nowUnix := int(time.Now().Unix())
	assert.Greater(t, deviceLastSeen, nowUnix-10)
	assert.Less(t, deviceLastSeen, nowUnix+10)
	assert.Greater(t, deviceFirstThrottled, nowUnix-10)
	assert.Less(t, deviceFirstThrottled, nowUnix+10)
}

type fakeLookup struct{ countryCode string }

func (l *fakeLookup) CountryCode(ip net.IP) string {
	return l.countryCode
}

func TestExpirationFor(t *testing.T) {
	timeZone := "Asia/Shanghai"
	tz, err := time.LoadLocation(timeZone)
	require.NoError(t, err)

	thursday := time.Date(2020, 12, 31, 23, 0, 0, 0, tz).In(time.UTC)
	friday := time.Date(2021, 1, 1, 0, 0, 0, 0, tz).Add(-1 * time.Nanosecond)
	sunday := time.Date(2021, 1, 3, 0, 0, 0, 0, tz).Add(-1 * time.Nanosecond)
	nextMonday := time.Date(2021, 1, 4, 0, 0, 0, 0, tz).Add(-1 * time.Nanosecond)

	require.Equal(t, friday.Unix(), expirationFor(thursday, throttle.Daily, timeZone), 0)
	require.Equal(t, friday.Unix(), expirationFor(thursday.Add(5*time.Minute), throttle.Daily, timeZone), 0)
	require.Equal(t, friday.Unix(), expirationFor(thursday, throttle.Monthly, timeZone), 0)
	require.Equal(t, friday.Unix(), expirationFor(thursday, throttle.Legacy, timeZone), 0)
	require.Equal(t, friday.Unix(), expirationFor(thursday.Add(5*time.Minute), throttle.Monthly, timeZone), 0)
	require.Equal(t, friday.Unix(), expirationFor(thursday.Add(5*time.Minute), throttle.Legacy, timeZone), 0)

	require.Equal(t, nextMonday.Unix(), expirationFor(thursday, throttle.Weekly, timeZone), 0)
	require.Equal(t, nextMonday.Unix(), expirationFor(thursday.Add(5*time.Minute), throttle.Weekly, timeZone), 0)
	require.Equal(t, nextMonday.Unix(), expirationFor(sunday, throttle.Weekly, timeZone), 0)
}
