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
	bl := New(50 * time.Millisecond)
	for i := 0; i < 10000; i++ {
		assert.True(t, bl.OnConnect(ip), "Should be able to continuously connect while succeeding")
		bl.Succeed(ip)
	}
}

func TestBlacklistFail(t *testing.T) {
	bl := New(5 * time.Millisecond)
	// Run through the same tests multiple times since this depends somewhat on timing
	for i := 0; i < 50; i++ {
		for j := 0; j < allowedFailures; j++ {
			time.Sleep(bl.maxIdleTime * 3)
			assert.True(t, bl.OnConnect(ip), "Should be able to continue connecting while failures are below threshold")
		}
		time.Sleep(bl.maxIdleTime * 3)
		assert.False(t, bl.OnConnect(ip), "Connecting should fail once failures exceed threshold")
		bl.successes <- ip
	}
}
