package throttle

import (
	"context"
	"crypto/tls"
	"strings"
	"testing"
	"time"

	"github.com/getlantern/golog/testlog"
	"github.com/getlantern/http-proxy-lantern/v2/internal/testutil"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/require"
)

const (
	refreshInterval = 10 * time.Millisecond

	deviceIDInSegment1    = "74" // this falls in segment 0.300786
	deviceIDInSegment2    = "78" // this falls in segment 0.914739
	deviceIDWithNoSegment = "55" // this falls in segment 0.016255

	goodSettings = `
{
	"default": {
		"default": [
			{"label": "cohort 1", "deviceFloor": 0.1, "deviceCeil": 0.5, "threshold": 1000, "rate": 100, "capResets": "weekly"},
			{"label": "cohort 2", "deviceFloor": 0.5, "deviceCeil": 1.0, "threshold": 1100, "rate": 110, "capResets": "monthly"}
		],
		"windows": [
			{"label": "cohort 3", "deviceFloor": 0.1, "deviceCeil": 0.5, "threshold": 2000, "rate": 200, "capResets": "weekly"},
			{"label": "cohort 4", "deviceFloor": 0.5, "deviceCeil": 1.0, "threshold": 2100, "rate": 210, "capResets": "monthly"}
		]
	},
	"cn": {
		"default": [
			{"label": "cohort 5", "deviceFloor": 0.1, "deviceCeil": 0.5, "threshold": 3000, "rate": 300, "capResets": "weekly"},
			{"label": "cohort 6", "deviceFloor": 0.5, "deviceCeil": 1.0, "threshold": 3100, "rate": 310, "capResets": "monthly"},
			{"label": "cohort 6", "deviceFloor": 0.5, "deviceCeil": 1.0, "threshold": 3200, "rate": 320, "capResets": "legacy"}
		],
		"windows": [
			{"label": "cohort 7", "deviceFloor": 0.1, "deviceCeil": 0.5, "threshold": 4000, "rate": 400, "capResets": "weekly"},
			{"label": "cohort 8", "deviceFloor": 0.5, "deviceCeil": 1.0, "threshold": 4100, "rate": 410, "capResets": "monthly"},
			{"label": "cohort 8", "deviceFloor": 0.5, "deviceCeil": 1.0, "threshold": 4200, "rate": 420, "capResets": "legacy"}
		]
	},
	"ir": {
		"default": [
			{"label": "capped", "threshold": 1000, "rate": 100, "capResets": "monthly"},
			{"label": "notcapped", "threshold": 1000000, "rate": 100, "capResets": "monthly", "appName": "specialapp"}
		]
	}
}`
)

func doTest(t *testing.T, cfg Config, deviceID, countryCode, platform, appName string, supportedDataCaps []string, expectedThreshold int64, expectedRate int64, expectedCapResets CapInterval, testCase string) {
	settings, ok := cfg.SettingsFor(deviceID, countryCode, platform, appName, supportedDataCaps)
	require.True(t, ok, "valid config for "+testCase)
	require.NotNil(t, settings, "non-nil settings for "+testCase)
	require.Equal(t, expectedThreshold, settings.Threshold, "correct threshold for "+testCase)
	require.Equal(t, expectedRate, settings.Rate, testCase, "correct rate for "+testCase)
	require.Equal(t, expectedCapResets, settings.CapResets, testCase, "correct ttl for "+testCase)
}

func TestThrottleConfig(t *testing.T) {
	stopCapture := testlog.Capture(t)
	defer stopCapture()

	rc := testutil.TestRedis(t)

	// try a bad config first
	require.NoError(t, rc.Set(context.Background(), "_throttle", "blah I'm bad settings blah", 0).Err())
	cfg := NewRedisConfig(rc, refreshInterval)
	_, ok := cfg.SettingsFor(deviceIDInSegment1, "cn", "windows", "lantern", []string{"monthly", "weekly"})
	require.False(t, ok, "Loading throttle settings from bad config should fail")

	// now do a good config
	require.NoError(t, rc.Set(context.Background(), "_throttle", goodSettings, 0).Err())
	cfg = NewRedisConfig(rc, refreshInterval)

	doTest(t, cfg, deviceIDInSegment1, "cn", "windows", "lantern", []string{"monthly", "weekly"}, 4000, 400, "weekly", "known country, known platform, segment 1")
	doTest(t, cfg, deviceIDInSegment2, "cn", "windows", "lantern", []string{"monthly", "weekly"}, 4100, 410, "monthly", "known country, known platform, segment 2")
	doTest(t, cfg, deviceIDInSegment1, "cn", "windows", "lantern", []string{"monthly", "weekly"}, 4000, 400, "weekly", "known country, known platform, unknown segment")
	doTest(t, cfg, deviceIDInSegment1, "cn", "windows", "lantern", nil, 4200, 420, "legacy", "known country, known platform, segment 1, legacy client")

	doTest(t, cfg, deviceIDInSegment1, "cn", "", "lantern", []string{"monthly", "weekly"}, 3000, 300, "weekly", "known country, unknown platform, segment 1")
	doTest(t, cfg, deviceIDInSegment2, "cn", "", "lantern", []string{"monthly", "weekly"}, 3100, 310, "monthly", "known country, unknown platform, segment 2")
	doTest(t, cfg, deviceIDInSegment1, "cn", "", "lantern", []string{"monthly", "weekly"}, 3000, 300, "weekly", "known country, unknown platform, unknown segment")

	doTest(t, cfg, deviceIDInSegment1, "de", "windows", "lantern", []string{"monthly", "weekly"}, 2000, 200, "weekly", "unknown country, known platform, segment 1")
	doTest(t, cfg, deviceIDInSegment2, "de", "windows", "lantern", []string{"monthly", "weekly"}, 2100, 210, "monthly", "unknown country, known platform, segment 2")
	doTest(t, cfg, deviceIDInSegment1, "de", "windows", "lantern", []string{"monthly", "weekly"}, 2000, 200, "weekly", "unknown country, known platform, unknown segment")

	doTest(t, cfg, deviceIDInSegment1, "de", "", "lantern", []string{"monthly", "weekly"}, 1000, 100, "weekly", "unknown country, unknown platform, segment 1")
	doTest(t, cfg, deviceIDInSegment2, "de", "", "lantern", []string{"monthly", "weekly"}, 1100, 110, "monthly", "unknown country, unknown platform, segment 2")
	doTest(t, cfg, deviceIDInSegment1, "de", "", "lantern", []string{"monthly", "weekly"}, 1000, 100, "weekly", "unknown country, unknown platform, unknown segment")

	doTest(t, cfg, deviceIDInSegment1, "ir", "", "lantern", []string{"monthly", "weekly"}, 1000, 100, "monthly", "capped named app")
	doTest(t, cfg, deviceIDInSegment1, "ir", "", "", []string{"monthly", "weekly"}, 1000, 100, "monthly", "capped unnamed app")
	doTest(t, cfg, deviceIDInSegment1, "ir", "", "specialapp", []string{"monthly", "weekly"}, 1000000, 100, "monthly", "uncapped app")

	// update settings
	require.NoError(t, rc.Set(context.Background(), "_throttle", strings.ReplaceAll(goodSettings, "4", "5"), 0).Err())
	time.Sleep(refreshInterval * 2)

	doTest(t, cfg, deviceIDInSegment1, "cn", "windows", "lantern", []string{"monthly", "weekly"}, 5000, 500, "weekly", "known country, known platform, segment 1, after update")
}

func TestForcedConfig(t *testing.T) {
	stopCapture := testlog.Capture(t)
	defer stopCapture()

	cfg := NewForcedConfig(1024, 512, "weekly")
	doTest(t, cfg, deviceIDInSegment1, "", "", "lantern", []string{"monthly", "weekly"}, 1024, 512, "weekly", "forced config")
}

func TestFailToConnectRedis(t *testing.T) {
	stopCapture := testlog.Capture(t)
	defer stopCapture()

	bogusClient := redis.NewClient(&redis.Options{
		Addr:      "localhost:80",
		TLSConfig: &tls.Config{InsecureSkipVerify: true},
	})

	cfg := NewRedisConfig(bogusClient, refreshInterval)
	_, ok := cfg.SettingsFor(deviceIDInSegment1, "cn", "windows", "lantern", []string{"monthly", "weekly"})
	require.False(t, ok, "Loading throttle settings when unable to contact redis should fail")

	redisClient := testutil.TestRedis(t)
	cfg = NewRedisConfig(redisClient, refreshInterval)
	require.NoError(t, redisClient.Set(context.Background(), "_throttle", goodSettings, 0).Err())

	time.Sleep(refreshInterval * 2)
	// Should load the config when Redis is back up online
	doTest(t, cfg, deviceIDInSegment1, "cn", "windows", "lantern", []string{"monthly", "weekly"}, 4000, 400, "weekly", "known country, known platform, segment 1, redis back online")
}
