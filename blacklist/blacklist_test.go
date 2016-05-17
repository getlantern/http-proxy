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
	bl := New(5*time.Millisecond, 2, 50*time.Millisecond)
	// Run through the same tests multiple times since this depends somewhat on timing
	for i := 0; i < 10; i++ {
		for j := 0; j < bl.allowedFailures; j++ {
			time.Sleep(bl.maxIdleTime * 3)
			assert.True(t, bl.OnConnect(ip), "Should be able to continue connecting while failures are below threshold")
		}
		time.Sleep(bl.maxIdleTime * 3)
		assert.False(t, bl.OnConnect(ip), "Connecting should fail once failures exceed threshold")

		time.Sleep(bl.blacklistExpiration * 2)
		assert.True(t, bl.OnConnect(ip), "Connecting after ip expired from blacklist should succeed")

		time.Sleep(bl.maxIdleTime / 2)
		bl.Succeed(ip)
	}
}
