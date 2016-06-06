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

// Options is a set of options to initialize a blacklist.
type Options struct {
	// The maximum amount of time we'll wait between the start of a connection
	// and seeing a successful HTTP request before we mark the connection as
	// failed.
	MaxIdleTime time.Duration
	// The maximum interval between two consecutive connect attempts from same
	// IP to make the IP as a counting target.
	MaxConnectInterval time.Duration
	// The number of consecutive failures allowed before an IP is blacklisted
	AllowedFailures int
	// How long an IP is allowed to remain on the blacklist.  In practice, an
	// IP may end up on the blacklist up to 1.1 * blacklistExpiration.
	Expiration time.Duration
}

// Blacklist is a blacklist of IPs.
type Blacklist struct {
	maxIdleTime         time.Duration
	maxConnectInterval  time.Duration
	allowedFailures     int
	blacklistExpiration time.Duration
	connections         chan string
	successes           chan string
	firstConnectionTime map[string]time.Time
	lastConnectionTime  map[string]time.Time
	failureCounts       map[string]int
	blacklist           map[string]time.Time
	mutex               sync.RWMutex
}

// New creates a new Blacklist with given options.
func New(opts Options) *Blacklist {
	bl := &Blacklist{
		maxIdleTime:         opts.MaxIdleTime,
		maxConnectInterval:  opts.MaxConnectInterval,
		allowedFailures:     opts.AllowedFailures,
		blacklistExpiration: opts.Expiration,
		connections:         make(chan string, 10000),
		successes:           make(chan string, 10000),
		firstConnectionTime: make(map[string]time.Time),
		lastConnectionTime:  make(map[string]time.Time),
		failureCounts:       make(map[string]int),
		blacklist:           make(map[string]time.Time),
	}
	go bl.track()
	return bl
}

// Succeed records a success for the given addr, which resets the failure count
// for that IP and removes it from the blacklist.
func (bl *Blacklist) Succeed(ip string) {
	select {
	case bl.successes <- ip:
		// ip submitted as success
	default:
		_ = log.Errorf("Unable to record success from %v", ip)
	}
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
		select {
		case bl.connections <- ip:
			// ip submitted as connected
		default:
			_ = log.Errorf("Unable to record connection from %v", ip)
		}
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
	now := time.Now()
	t, exists := bl.lastConnectionTime[ip]
	bl.lastConnectionTime[ip] = now
	if now.Sub(t) > bl.maxConnectInterval {
		bl.failureCounts[ip] = 0
		return
	}

	_, exists = bl.firstConnectionTime[ip]
	if !exists {
		bl.firstConnectionTime[ip] = now
	}
}

func (bl *Blacklist) onSuccess(ip string) {
	bl.failureCounts[ip] = 0
	delete(bl.lastConnectionTime, ip)
	delete(bl.firstConnectionTime, ip)
	bl.mutex.Lock()
	delete(bl.blacklist, ip)
	bl.mutex.Unlock()
}

func (bl *Blacklist) checkForIdlers() {
	log.Trace("Checking for idlers")
	now := time.Now()
	var blacklistAdditions []string
	for ip, t := range bl.firstConnectionTime {
		if now.Sub(t) > bl.maxIdleTime {
			log.Debugf("%v connected but failed to successfully send an HTTP request within %v", ip, bl.maxIdleTime)
			delete(bl.firstConnectionTime, ip)

			count := bl.failureCounts[ip] + 1
			bl.failureCounts[ip] = count
			if count >= bl.allowedFailures {
				_ = log.Errorf("Blacklisting %v", ip)
				blacklistAdditions = append(blacklistAdditions, ip)
			}
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
