package blacklist

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBlacklist(t *testing.T) {
	ip := "8.8.8.8"
	for i := 0; i < allowedFailures-1; i++ {
		Fail(ip)
		time.Sleep(25 * time.Millisecond)
		assert.True(t, IsNotBlacklisted(ip), "IP shouldn't be blacklisted yet")
	}
	Succeed(ip)
	for i := 0; i < allowedFailures-1; i++ {
		Fail(ip)
		time.Sleep(25 * time.Millisecond)
		assert.True(t, IsNotBlacklisted(ip), "IP shouldn't be blacklisted yet")
	}
	Fail(ip)
	time.Sleep(25 * time.Millisecond)
	assert.False(t, IsNotBlacklisted(ip), "IP should be blacklisted now")
	Succeed(ip)
	time.Sleep(25 * time.Millisecond)
	assert.True(t, IsNotBlacklisted(ip), "IP shouldn't be blacklisted anymore")
}
