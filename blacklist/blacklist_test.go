package blacklist

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const (
	ip = "8.8.8.8"
)

func init() {
	blacklistingEnabled = true
}

func TestBlacklistSucceed(t *testing.T) {
	bl := New(Options{50 * time.Millisecond, 1 * time.Millisecond, 2, 5 * time.Second})
	for i := 0; i < 10000; i++ {
		assert.True(t, bl.OnConnect(ip), "Should be able to continuously connect while succeeding")
		bl.Succeed(ip)
	}
}

func TestBlacklistFail(t *testing.T) {
	maxIdleTime := 10 * time.Millisecond
	bl := New(Options{
		MaxIdleTime:        maxIdleTime,
		MaxConnectInterval: maxIdleTime * 5,
		AllowedFailures:    3,
		Expiration:         maxIdleTime * 50,
	})
	// Run through the same tests multiple times since this depends somewhat on timing
	for i := 0; i < 10; i++ {
		for j := 0; j < bl.allowedFailures; j++ {
			assert.True(t, bl.OnConnect(ip), "Should be able to continue connecting while failures are below threshold")
			time.Sleep(bl.maxConnectInterval * 2)
		}
		assert.True(t, bl.OnConnect(ip), "Connecting should not fail if the interval between each connect attempt is long enough")

		for k := 0; k < bl.allowedFailures; k++ {
			assert.True(t, bl.OnConnect(ip), "Should be able to continue connecting while failures are below threshold")
			time.Sleep(bl.maxIdleTime * 3)
		}
		// Connect a couple more times to make sure we exceed threshold
		for k := 0; k < bl.allowedFailures; k++ {
			bl.OnConnect(ip)
		}
		assert.False(t, bl.OnConnect(ip), "Connecting should fail once failures exceed threshold")

		time.Sleep(bl.blacklistExpiration * 2)
		assert.True(t, bl.OnConnect(ip), "Connecting after ip expired from blacklist should succeed")

		time.Sleep(bl.maxIdleTime / 2)
		bl.Succeed(ip)
	}
}
