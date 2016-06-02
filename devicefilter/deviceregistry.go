package devicefilter

import (
	"sync"
)

var (
	mtx     sync.RWMutex
	devices map[string]bool
)

func init() {
	DeviceRegistryReset()
}

func DeviceRegistryAdd(dev string) {
	mtx.Lock()
	devices[dev] = true
	mtx.Unlock()
}

func DeviceRegistryRemove(dev string) {
	mtx.Lock()
	delete(devices, dev)
	mtx.Unlock()
}

func DeviceRegistryExists(dev string) bool {
	mtx.RLock()
	_, ok := devices[dev]
	mtx.RUnlock()
	return ok
}

func DeviceRegistryReset() {
	mtx.Lock()
	devices = make(map[string]bool)
	mtx.Unlock()
}
