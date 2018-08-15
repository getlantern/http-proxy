package throttle

import (
	"testing"
	"time"

	"github.com/getlantern/testredis"
	"github.com/stretchr/testify/assert"
)

const (
	refreshInterval = 10 * time.Millisecond

	desktopDeviceID = "12345678"
	mobileDeviceID  = "123456789"
)

func doTest(t *testing.T, cfg Config, deviceID string, countryCode string, expectedThreshold int64, expectedRate int64, validConfig bool) {
	threshold, rate, ok := cfg.ThresholdAndRateFor(deviceID, countryCode)
	assert.EqualValues(t, expectedThreshold, threshold)
	assert.EqualValues(t, expectedRate, rate)
	assert.Equal(t, ok, validConfig)
}

func TestThrottleConfig(t *testing.T) {
	r, err := testredis.Open()
	if !assert.NoError(t, err) {
		return
	}
	defer r.Close()

	rc := r.Client()
	defer rc.Close()

	if !assert.NoError(t, rc.HMSet("_throttle:desktop", map[string]string{
		DefaultCountryCode: "60|6",
		"cn":               "50|5"}).Err()) {
		return
	}
	if !assert.NoError(t, rc.HMSet("_throttle:mobile", map[string]string{
		DefaultCountryCode: "40|4",
		"cn":               "30|3"}).Err()) {
		return
	}

	cfg := NewRedisConfig(rc, refreshInterval)

	doTest(t, cfg, desktopDeviceID, "cn", 50, 5, true)
	doTest(t, cfg, desktopDeviceID, "us", 60, 6, true)
	doTest(t, cfg, desktopDeviceID, "", 60, 6, true)

	doTest(t, cfg, mobileDeviceID, "cn", 30, 3, true)
	doTest(t, cfg, mobileDeviceID, "us", 40, 4, true)
	doTest(t, cfg, mobileDeviceID, "", 40, 4, true)

	// update settings
	if !assert.NoError(t, rc.HMSet("_throttle:desktop", map[string]string{
		DefaultCountryCode: "600|60",
		"cn":               "500|50",
		"bl":               "asdfadsf",
		"bt":               "adsfadsfd|10",
		"br":               "1000000|asdfd"}).Err()) {
		return
	}
	if !assert.NoError(t, rc.HMSet("_throttle:mobile", map[string]string{
		DefaultCountryCode: "400|40",
		"cn":               "300|30"}).Err()) {
		return
	}
	time.Sleep(refreshInterval * 2)

	doTest(t, cfg, desktopDeviceID, "cn", 500, 50, true)
	doTest(t, cfg, desktopDeviceID, "us", 600, 60, true)
	doTest(t, cfg, desktopDeviceID, "bl", 600, 60, true)
	doTest(t, cfg, desktopDeviceID, "bt", 600, 60, true)
	doTest(t, cfg, desktopDeviceID, "br", 600, 60, true)
	doTest(t, cfg, desktopDeviceID, "", 600, 60, true)

	doTest(t, cfg, mobileDeviceID, "cn", 300, 30, true)
	doTest(t, cfg, mobileDeviceID, "us", 400, 40, true)
	doTest(t, cfg, mobileDeviceID, "bl", 400, 40, true)
	doTest(t, cfg, mobileDeviceID, "bt", 400, 40, true)
	doTest(t, cfg, mobileDeviceID, "br", 400, 40, true)
	doTest(t, cfg, mobileDeviceID, "", 400, 40, true)

	if !assert.NoError(t, rc.HDel("_throttle:desktop", "us", "__").Err()) {
		return
	}
	time.Sleep(refreshInterval * 2)

	doTest(t, cfg, desktopDeviceID, "cn", 500, 50, true)
	doTest(t, cfg, desktopDeviceID, "us", 0, 0, false)
	doTest(t, cfg, desktopDeviceID, "", 0, 0, false)

	doTest(t, cfg, mobileDeviceID, "cn", 300, 30, true)
	doTest(t, cfg, mobileDeviceID, "us", 400, 40, true)
	doTest(t, cfg, mobileDeviceID, "", 400, 40, true)
}

func TestForcedConfig(t *testing.T) {
	cfg := NewForcedConfig(1024, 512)
	doTest(t, cfg, mobileDeviceID, "", 1024, 512, true)
	doTest(t, cfg, desktopDeviceID, "", 1024, 512, true)
	doTest(t, cfg, mobileDeviceID, "cn", 1024, 512, true)
	doTest(t, cfg, desktopDeviceID, "cn", 1024, 512, true)
	doTest(t, cfg, mobileDeviceID, "bl", 1024, 512, true)
	doTest(t, cfg, desktopDeviceID, "bl", 1024, 512, true)
}

func TestFailToConnectRedis(t *testing.T) {
	r, err := testredis.OpenUnstarted()
	if !assert.NoError(t, err) {
		return
	}

	rc := r.Client()
	defer rc.Close()

	cfg := NewRedisConfig(rc, refreshInterval)

	doTest(t, cfg, desktopDeviceID, "cn", 0, 0, false)
	doTest(t, cfg, desktopDeviceID, "us", 0, 0, false)
	doTest(t, cfg, desktopDeviceID, "", 0, 0, false)

	r.Start()
	defer r.Close()

	if !assert.NoError(t, rc.HMSet("_throttle:desktop", map[string]string{
		DefaultCountryCode: "60|6",
	}).Err()) {
		return
	}

	time.Sleep(refreshInterval * 2)
	// Should load the config when Redis is back up online
	doTest(t, cfg, desktopDeviceID, "any", 60, 6, true)
}
