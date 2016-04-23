// Package blacklist provides a mechanism for blacklisting IP addresses that
// connect but never make it past our security filtering, either because they're
// not sending HTTP requests or sending invalid HTTP requests.
package blacklist

import (
	"sync"
	"time"

	"github.com/getlantern/golog"
)

var (
	log = golog.LoggerFor("blacklist")
)

// Blacklist is a blacklist of IPs.
type Blacklist struct {
	maxIdleTime         time.Duration
	allowedFailures     int
	blacklistExpiration time.Duration
	connections         chan string
	successes           chan string
	firstConnectionTime map[string]time.Time
	failureCounts       map[string]int
	blacklist           map[string]time.Time
	mutex               sync.RWMutex
}

// New creates a new Blacklist.
// maxIdleTime - the maximum amount of time we'll wait between the start of a
//               connection and seeing a successful HTTP request before we mark
//               the connection as failed.
// allowedFailures - the number of consecutive failures allowed before an IP is
//                   blacklisted
// blacklistExpiration - how long an IP is allowed to remain on the blacklist.
//                       In practice, an IP may end up on the blacklist up to
//                       1.1 * blacklistExpiration.
func New(maxIdleTime time.Duration, allowedFailures int, blacklistExpiration time.Duration) *Blacklist {
	bl := &Blacklist{
		maxIdleTime:         maxIdleTime,
		allowedFailures:     allowedFailures,
		blacklistExpiration: blacklistExpiration,
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
	idleTimer := time.NewTimer(bl.maxIdleTime)
	blacklistTimer := time.NewTimer(bl.blacklistExpiration / 10)
	for {
		select {
		case ip := <-bl.connections:
			bl.onConnection(ip)
		case ip := <-bl.successes:
			bl.onSuccess(ip)
		case <-idleTimer.C:
			bl.checkForIdlers()
			idleTimer.Reset(bl.maxIdleTime)
		case <-blacklistTimer.C:
			bl.checkExpiration()
			blacklistTimer.Reset(bl.blacklistExpiration / 10)
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
			if count >= bl.allowedFailures {
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

func (bl *Blacklist) checkExpiration() {
	now := time.Now()
	bl.mutex.Lock()
	for ip, blacklistedAt := range bl.blacklist {
		if now.Sub(blacklistedAt) > bl.blacklistExpiration {
			log.Debugf("Removing %v from blacklist", ip)
			delete(bl.blacklist, ip)
			delete(bl.failureCounts, ip)
			delete(bl.firstConnectionTime, ip)
		}
	}
	bl.mutex.Unlock()
}
