package redis

import (
	"testing"
	"time"

	"github.com/getlantern/testify/assert"
)

func TestItem(t *testing.T) {
	item := &Item{val: 0}
	assert.True(t, item.expired(), "Expected item to be expired by default")

	expiration := time.Now().Add(time.Second)
	item.expires = &expiration
	assert.False(t, item.expired(), "Expected item to not be expired")

	expiration = time.Now().Add(0 - time.Second)
	item.expires = &expiration
	assert.True(t, item.expired(), "Expected item to be expired once time has passed")

	item.touch(time.Second)
	assert.False(t, item.expired(), "Expected item to not be expired once touched")
}

func TestIncr(t *testing.T) {
	cache := &TTLCache{
		ttl:   time.Millisecond * 50,
		items: map[string]*Item{},
	}

	val, ok := cache.Incr("mykey")

	assert.False(t, ok, "Cache key should NOT have been found")
	assert.Equal(t, uint64(1), val, "Key should be 1")

	val, ok = cache.Incr("mykey")

	assert.True(t, ok, "Cache key should have been found")
	assert.Equal(t, uint64(2), val, "Key should have been incremented by 1")

	// Expire and increment

	time.Sleep(time.Millisecond * 50)

	val, ok = cache.Incr("mykey")

	assert.False(t, ok, "Cache key should NOT have been found")
	assert.Equal(t, uint64(1), val, "Expired key should be 1 after increment")

	val, ok = cache.Incr("mykey")

	assert.Equal(t, uint64(2), val, "Expired key should be 2 after increment")
}
