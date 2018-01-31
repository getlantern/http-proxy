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

	cfg, err := NewRedisConfig(rc, refreshInterval, 0, 0)
	if !assert.NoError(t, err) {
		return
	}

	doTest := func(deviceID string, countryCode string, expectedThreshold int64, expectedRate int64) {
		threshold, rate := cfg.ThresholdAndRateFor(deviceID, countryCode)
		assert.EqualValues(t, expectedThreshold, threshold)
		assert.EqualValues(t, expectedRate, rate)
	}

	doTest(desktopDeviceID, "cn", 50, 5)
	doTest(desktopDeviceID, "us", 60, 6)
	doTest(desktopDeviceID, "", 60, 6)

	doTest(mobileDeviceID, "cn", 30, 3)
	doTest(mobileDeviceID, "us", 40, 4)
	doTest(mobileDeviceID, "", 40, 4)

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

	doTest(desktopDeviceID, "cn", 500, 50)
	doTest(desktopDeviceID, "us", 600, 60)
	doTest(desktopDeviceID, "bl", 600, 60)
	doTest(desktopDeviceID, "bt", 600, 60)
	doTest(desktopDeviceID, "br", 600, 60)
	doTest(desktopDeviceID, "", 600, 60)

	doTest(mobileDeviceID, "cn", 300, 30)
	doTest(mobileDeviceID, "us", 400, 40)
	doTest(mobileDeviceID, "bl", 400, 40)
	doTest(mobileDeviceID, "bt", 400, 40)
	doTest(mobileDeviceID, "br", 400, 40)
	doTest(mobileDeviceID, "", 400, 40)
}
