// Package blacklist provides a mechanism for blacklisting IP addresses that
// connect but never make it past our security filtering, either because they're
// not sending HTTP requests or sending invalid HTTP requests.
package blacklist

import (
	"sync"
	"time"

	"github.com/getlantern/golog"
)

const (
	allowedFailures = 3
)

var (
	log = golog.LoggerFor("blacklist")
)

// Blacklist is a blacklist of IPs.
type Blacklist struct {
	maxIdleTime         time.Duration
	connections         chan string
	successes           chan string
	firstConnectionTime map[string]time.Time
	failureCounts       map[string]int
	blacklist           map[string]time.Time
	mutex               sync.RWMutex
}

// New creates a new Blacklist with the given maxIdleTime.
func New(maxIdleTime time.Duration) *Blacklist {
	bl := &Blacklist{
		maxIdleTime:         maxIdleTime,
		connections:         make(chan string, 1000),
		successes:           make(chan string, 1000),
		firstConnectionTime: make(map[string]time.Time),
		failureCounts:       make(map[string]int),
		blacklist:           make(map[string]time.Time),
	}
	go bl.track()
	return bl
}

// Succeed records a success for the given addr, which resets the failure count
// for that IP and removes it from the blacklist.
func (bl *Blacklist) Succeed(ip string) {
	bl.successes <- ip
}

// OnConnect records an attempt to connect from the given IP. If the IP is
// blacklisted, this returns false.
func (bl *Blacklist) OnConnect(ip string) bool {
	bl.mutex.RLock()
	defer bl.mutex.RUnlock()
	_, blacklisted := bl.blacklist[ip]
	if blacklisted {
		log.Debugf("%v is blacklisted", ip)
	} else {
		bl.connections <- ip
	}
	return !blacklisted
}

func (bl *Blacklist) track() {
	timer := time.NewTimer(bl.maxIdleTime)
	for {
		select {
		case ip := <-bl.connections:
			bl.onConnection(ip)
		case ip := <-bl.successes:
			bl.onSuccess(ip)
		case <-timer.C:
			bl.checkForIdlers()
			timer.Reset(bl.maxIdleTime)
		}
	}
}

func (bl *Blacklist) onConnection(ip string) {
	_, exists := bl.firstConnectionTime[ip]
	if !exists {
		bl.firstConnectionTime[ip] = time.Now()
	}
}

func (bl *Blacklist) onSuccess(ip string) {
	bl.failureCounts[ip] = 0
	delete(bl.firstConnectionTime, ip)
	bl.mutex.Lock()
	delete(bl.blacklist, ip)
	bl.mutex.Unlock()
}

func (bl *Blacklist) checkForIdlers() {
	log.Debug("Checking for idlers")
	now := time.Now()
	for ip, t := range bl.firstConnectionTime {
		var blacklistAdditions []string
		if now.Sub(t) > bl.maxIdleTime {
			log.Debugf("%v connected but failed to successfully send an HTTP request within %v", ip, bl.maxIdleTime)
			delete(bl.firstConnectionTime, ip)
			count := bl.failureCounts[ip] + 1
			bl.failureCounts[ip] = count
			if count >= allowedFailures {
				log.Debugf("Blacklisting %v", ip)
				blacklistAdditions = append(blacklistAdditions, ip)
			}
		}
		if len(blacklistAdditions) > 0 {
			bl.mutex.Lock()
			for _, ip := range blacklistAdditions {
				bl.blacklist[ip] = now
			}
			bl.mutex.Unlock()
		}
	}
}
