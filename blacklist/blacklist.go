// Package blacklist provides a mechanism for blacklisting IP addresses
package blacklist

import (
	"sync"
	"time"

	"github.com/getlantern/golog"
)

const (
	allowedFailures = 10
)

var (
	log = golog.LoggerFor("blacklist")

	failures      = make(chan string, 1000)
	successes     = make(chan string, 1000)
	failureCounts = make(map[string]int)
	blacklist     = make(map[string]time.Time)
	mutex         sync.RWMutex
)

func init() {
	go track()
}

// Fail records a failure for the given addr
func Fail(addr string) {
	failures <- addr
}

// Succeed records a success for the given addr, which resets the failure count for that ip and removes it from the blacklist.
func Succeed(addr string) {
	successes <- addr
}

// IsNotBlacklisted checks whether the remote address is not blacklisted.
func IsNotBlacklisted(addr string) bool {
	mutex.RLock()
	defer mutex.RUnlock()
	_, blacklisted := blacklist[addr]
	if blacklisted {
		log.Debugf("%v is blacklisted", addr)
	}
	return !blacklisted
}

func track() {
	for {
		select {
		case addr := <-failures:
			count := failureCounts[addr] + 1
			failureCounts[addr] = count
			if count >= allowedFailures {
				log.Debugf("Blacklisting %v", addr)
				mutex.Lock()
				blacklist[addr] = time.Now()
				mutex.Unlock()
			}
		case addr := <-successes:
			failureCounts[addr] = 0
			mutex.Lock()
			delete(blacklist, addr)
			mutex.Unlock()
		}
	}
}
