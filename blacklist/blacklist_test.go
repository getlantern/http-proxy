package blacklist

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const (
	ip = "8.8.8.8"
)

func TestBlacklistSucceed(t *testing.T) {
	bl := New(50*time.Millisecond, 2, 5*time.Second)
	for i := 0; i < 10000; i++ {
		assert.True(t, bl.OnConnect(ip), "Should be able to continuously connect while succeeding")
		bl.Succeed(ip)
	}
}

func TestBlacklistFail(t *testing.T) {
	bl := New(5*time.Millisecond, 2, 25*time.Millisecond)
	// Run through the same tests multiple times since this depends somewhat on timing
	for i := 0; i < 50; i++ {
		for j := 0; j < bl.allowedFailures-1; j++ {
			time.Sleep(bl.maxIdleTime * 3)
			assert.True(t, bl.OnConnect(ip), "Should be able to continue connecting while failures are below threshold")
		}
		// The next failure would usually cause us to get blacklisted, but we'll wait longer than the reset interval
		time.Sleep(bl.failureResetInterval * 2)

		for j := 0; j < bl.allowedFailures; j++ {
			time.Sleep(bl.maxIdleTime * 3)
			assert.True(t, bl.OnConnect(ip), "Should be able to continue connecting while failures are below threshold")
		}
		time.Sleep(bl.maxIdleTime * 3)
		assert.False(t, bl.OnConnect(ip), "Connecting should fail once failures exceed threshold")
		bl.Succeed(ip)
	}
}
