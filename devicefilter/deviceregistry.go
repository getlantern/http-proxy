package devicefilter

import (
	"sync"
)

type DeviceRegistry struct {
	sync.RWMutex
	devices map[string]bool
}

var (
	globalRegistry *DeviceRegistry
)

func init() {
	globalRegistry = &DeviceRegistry{
		devices: make(map[string]bool),
	}
}

func DeviceRegistryAdd(dev string) {
	globalRegistry.Lock()
	globalRegistry.devices[dev] = true
	globalRegistry.Unlock()
}

func DeviceRegistryExists(dev string) bool {
	globalRegistry.RLock()
	_, ok := globalRegistry.devices[dev]
	globalRegistry.RUnlock()
	return ok
}

func DeviceRegistryReset() {
	globalRegistry.Lock()
	globalRegistry.devices = make(map[string]bool)
	globalRegistry.Unlock()
}
