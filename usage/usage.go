package usage

import (
	"sync"
	"time"
)

var (
	mutex           sync.RWMutex
	usageByDeviceID = make(map[string]*Usage)
)

type Usage struct {
	Bytes uint64
	AsOf  time.Time
}

// Set sets the Usage in bytes for the given device as of the given time.
func Set(dev string, usage uint64, asOf time.Time) {
	mutex.Lock()
	usageByDeviceID[dev] = &Usage{usage, asOf}
	mutex.Unlock()
}

// Get gets the Usage for the given device.
func Get(dev string) *Usage {
	mutex.RLock()
	result := usageByDeviceID[dev]
	mutex.RUnlock()
	return result
}
