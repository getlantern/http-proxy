// TTL cache for counters. Based on https://github.com/wunderlist/ttlcache
package redis

import (
	"sync"
	"sync/atomic"
	"time"
)

type Item struct {
	sync.RWMutex
	val     uint64
	expires *time.Time
}

func (item *Item) touch(duration time.Duration) {
	item.Lock()
	expiration := time.Now().Add(duration)
	item.expires = &expiration
	item.Unlock()
}

func (item *Item) expired() bool {
	var value bool
	item.RLock()
	if item.expires == nil {
		value = true
	} else {
		value = item.expires.Before(time.Now())
	}
	item.RUnlock()
	return value
}

type TTLCache struct {
	mutex sync.Mutex
	ttl   time.Duration
	items map[string]*Item
}

func (cache *TTLCache) Incr(key string) (val uint64, found bool) {
	cache.mutex.Lock()
	defer cache.mutex.Unlock()

	item, exists := cache.items[key]
	if !exists {
		val = uint64(1)
		item := &Item{val: val}
		item.touch(cache.ttl)
		cache.items[key] = item

		return val, false
	}

	if item.expired() {
		val = uint64(1)
		item.touch(cache.ttl)
		atomic.StoreUint64(&item.val, val)

		return val, false
	}

	item.touch(cache.ttl)
	val = atomic.AddUint64(&item.val, 1)

	return val, true
}

func (cache *TTLCache) cleanup() {
	cache.mutex.Lock()
	for key, item := range cache.items {
		if item.expired() {
			delete(cache.items, key)
		}
	}
	cache.mutex.Unlock()
}

func (cache *TTLCache) startCleanupTimer() {
	duration := cache.ttl
	if duration < time.Second {
		duration = time.Second
	}
	ticker := time.Tick(duration)
	go (func() {
		for {
			select {
			case <-ticker:
				cache.cleanup()
			}
		}
	})()
}

// NewTTLCache is a helper to create instance of the TTLCache struct
func NewTTLCache(duration time.Duration) *TTLCache {
	cache := &TTLCache{
		ttl:   duration,
		items: map[string]*Item{},
	}
	cache.startCleanupTimer()
	return cache
}
