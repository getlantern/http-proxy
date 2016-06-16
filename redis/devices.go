package redis

import (
	"strconv"
	"sync"
	"time"

	"github.com/getlantern/http-proxy-lantern/usage"
	"gopkg.in/redis.v3"
)

type ongoingSet struct {
	set map[string]bool
	sync.RWMutex
}

func (s *ongoingSet) add(dev string) {
	s.Lock()
	s.set[dev] = true
	s.Unlock()
}

func (s *ongoingSet) del(dev string) {
	s.Lock()
	delete(s.set, dev)
	s.Unlock()
}

func (s *ongoingSet) isMember(dev string) bool {
	s.RLock()
	_, ok := s.set[dev]
	s.RUnlock()
	return ok
}

type DeviceFetcher struct {
	rc      *redis.Client
	ongoing ongoingSet
	queue   chan string
}

func NewDeviceFetcher(rc *redis.Client) *DeviceFetcher {
	df := &DeviceFetcher{
		rc:    rc,
		queue: make(chan string),
	}

	go func() {
		for dev := range df.queue {
			df.retrieveDeviceUsage(dev)
		}
	}()

	return df
}

func (df *DeviceFetcher) RequestNewDeviceUsage(device string) {
	if df.ongoing.isMember(device) {
		return
	} else {
		df.ongoing.add(device)
	}
}

func (df *DeviceFetcher) retrieveDeviceUsage(device string) error {
	vals, err := df.rc.HMGet("_client:"+device, "bytesIn", "bytesOut").Result()
	if err != nil {
		return err
	} else if vals[0] == nil || vals[1] == nil {
		// No entry found or partially stored, nothing to be done
		return nil
	}

	bytesIn, err := strconv.ParseUint(vals[0].(string), 10, 64)
	if err != nil {
		return err
	}
	bytesOut, err := strconv.ParseUint(vals[1].(string), 10, 64)
	if err != nil {
		return err
	}

	usage.Set(device, bytesIn+bytesOut, time.Now())
	df.ongoing.del(device)
	return nil
}
