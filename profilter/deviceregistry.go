package profilter

import (
	"sync"
	"sync/atomic"
)

type DevicesMap map[uint64]map[string]bool

type DeviceRegistry struct {
	devicesPerUser atomic.Value
	womutex        sync.Mutex
}

func NewDeviceRegistry() *DeviceRegistry {
	return &DeviceRegistry{}
}

func (r *DeviceRegistry) AddUserDevice(userID uint64, device string) {
	r.womutex.Lock()
	defer r.womutex.Unlock()

	devicesPerUser := r.devicesPerUser.Load().(DevicesMap)
	// Deep-copy the nested maps
	newDevicesPerUser := make(DevicesMap)
	for k, v := range devicesPerUser {
		newDevices := make(map[string]bool)
		for k2, v2 := range v {
			newDevices[k2] = v2
		}
		newDevicesPerUser[k] = newDevices
	}
	newDevicesPerUser[userID][device] = true
	r.devicesPerUser.Store(newDevicesPerUser)
}

func (r *DeviceRegistry) GetUserDevices(userID uint64) map[string]bool {
	devices := r.devicesPerUser.Load().(DevicesMap)
	return devices[userID]
}

func (r *DeviceRegistry) SetUserDevices(newDevicesPerUser DevicesMap) {
	r.devicesPerUser.Store(newDevicesPerUser)
}
