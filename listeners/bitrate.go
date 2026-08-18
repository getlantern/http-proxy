package listeners

import (
	"net"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/getlantern/ratelimit"
)

const (
	minSleep = 5 * time.Millisecond // don't bother sleeping for less than this amount of time
)

// rateBuckets pairs the token buckets with the rates they were built from so a
// re-rate swaps both together and a reader never sees a bucket that disagrees
// with its rate.
type rateBuckets struct {
	r         *ratelimit.Bucket
	w         *ratelimit.Bucket
	rateRead  int64
	rateWrite int64
}

func newRateBuckets(rateRead, rateWrite int64) *rateBuckets {
	b := &rateBuckets{rateRead: rateRead, rateWrite: rateWrite}
	if rateRead > 0 {
		b.r = ratelimit.NewBucketWithRate(float64(rateRead), rateRead)
	}
	if rateWrite > 0 {
		b.w = ratelimit.NewBucketWithRate(float64(rateWrite), rateWrite)
	}
	return b
}

// RateLimiter caps read and write throughput on the connections it is attached
// to. Its rates are mutable (see SetRates): one limiter shared by every
// connection of a device can be re-rated in place, and the new rate applies to
// connections that are already open rather than only to the next one.
type RateLimiter struct {
	buckets atomic.Pointer[rateBuckets]
}

func NewRateLimiter(rateRead, rateWrite int64) *RateLimiter {
	l := &RateLimiter{}
	l.buckets.Store(newRateBuckets(rateRead, rateWrite))
	return l
}

// SetRates re-rates the limiter. It is a no-op when the rates are unchanged, so
// a caller refreshing on every reporting cycle does not continually reset the
// token buckets.
func (l *RateLimiter) SetRates(rateRead, rateWrite int64) {
	if cur := l.buckets.Load(); cur.rateRead == rateRead && cur.rateWrite == rateWrite {
		return
	}
	l.buckets.Store(newRateBuckets(rateRead, rateWrite))
}

func (l *RateLimiter) GetRateRead() int64 {
	return l.buckets.Load().rateRead
}

func (l *RateLimiter) GetRateWrite() int64 {
	return l.buckets.Load().rateWrite
}

func (b *rateBuckets) waitRead(n int) {
	if b.r == nil {
		return
	}
	if d := b.r.Take(int64(n)); d > 0 {
		sleep(d)
	}
}

func (b *rateBuckets) waitWrite(n int) {
	if b.w == nil {
		return
	}
	if d := b.w.Take(int64(n)); d > 0 {
		sleep(d)
	}
}

// In order to avoid lots of very short (and relatively expensive) sleeps, never sleep for
// less than minSleep.
func sleep(d time.Duration) {
	if d < minSleep {
		d = minSleep
	}
	time.Sleep(d)
}

type bitrateListener struct {
	net.Listener
}

func NewBitrateListener(l net.Listener) net.Listener {
	return &bitrateListener{l}
}

// unlimited is the limiter every conn starts with until a filter attaches a
// real one. A (0,0) limiter never waits and nothing ever re-rates it —
// ControlMessage replaces the pointer wholesale — so one shared instance
// serves every conn instead of allocating two objects per accept.
var unlimited = NewRateLimiter(0, 0)

func (bl *bitrateListener) Accept() (net.Conn, error) {
	c, err := bl.Listener.Accept()
	if err != nil {
		return nil, err
	}

	wc, _ := c.(WrapConnEmbeddable)
	brc := &bitrateConn{
		WrapConnEmbeddable: wc,
		Conn:               c,
	}
	brc.limiter.Store(unlimited)
	return brc, err
}

// Bitrate Conn wrapper
type bitrateConn struct {
	WrapConnEmbeddable
	net.Conn
	limiter atomic.Pointer[RateLimiter]
}

func (c *bitrateConn) Read(p []byte) (n int, err error) {
	b := c.limiter.Load().buckets.Load()
	if b.rateRead == 0 {
		return c.Conn.Read(p)
	}

	n, err = c.Conn.Read(p)
	if err == nil {
		b.waitRead(n)
	}
	return
}

func (c *bitrateConn) Write(p []byte) (n int, err error) {
	b := c.limiter.Load().buckets.Load()
	if b.rateWrite == 0 {
		return c.Conn.Write(p)
	}

	n, err = c.Conn.Write(p)
	if err == nil {
		b.waitWrite(n)
	}
	return
}

func (c *bitrateConn) OnState(s http.ConnState) {
	// Pass down to wrapped connections
	if c.WrapConnEmbeddable != nil {
		c.WrapConnEmbeddable.OnState(s)
	}
}

func (c *bitrateConn) ControlMessage(msgType string, data interface{}) {
	// per user message always overrides the active flag
	if msgType == "throttle" {
		c.limiter.Store(data.(*RateLimiter))
	}

	if c.WrapConnEmbeddable != nil {
		c.WrapConnEmbeddable.ControlMessage(msgType, data)
	}
}

func (c *bitrateConn) Wrapped() net.Conn {
	return c.Conn
}
