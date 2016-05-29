package usage

import (
	"sync"
)

var (
	mutex           sync.RWMutex
	usageByDeviceID = make(map[string]uint64)
)

// Set sets the usage in bytes for the given device.
func Set(dev string, usage uint64) {
	mutex.Lock()
	usageByDeviceID[dev] = usage
	mutex.Unlock()
}

// Get gets the usage in bytes for the given device.
func Get(dev string) uint64 {
	mutex.RLock()
	result := usageByDeviceID[dev]
	mutex.RUnlock()
	return result
}
