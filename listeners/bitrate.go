package listeners

import (
	"net"
	"net/http"
	"time"

	"github.com/getlantern/http-proxy/listeners"
	"github.com/juju/ratelimit"
)

const (
	// prefer to read or write at least this number of bytes
	// at once if possible.
	preferredMinIO = 512
)

type RateLimiter struct {
	r    *ratelimit.Bucket
	w    *ratelimit.Bucket
	rate int64
}

func NewRateLimiter(rate int64) *RateLimiter {
	l := &RateLimiter{
		rate: rate,
	}
	if rate > 0 {
		l.r = ratelimit.NewBucketWithRate(float64(rate), rate)
		l.w = ratelimit.NewBucketWithRate(float64(rate), rate)
	}
	return l
}

// Acquire up to max read tokens. Will return as soon as
// between min and max reads are acquired. min is the
// lesser of preferredMinIO and max
func (l *RateLimiter) takeRead(max int64) int64 {
	return l.take(l.r, max)
}

// Acquire up to max write tokens. Will return as soon as
// between min and max writes are acquired. min is the
// lesser of preferredMinIO and max
func (l *RateLimiter) takeWrite(max int64) int64 {
	return l.take(l.w, max)
}

func (l *RateLimiter) take(b *ratelimit.Bucket, max int64) int64 {
	if b == nil {
		return max
	}

	min := int64(preferredMinIO)
	if max < min {
		min = max
	}

	d := b.Take(min)
	taken := min
	if d > 0 {
		time.Sleep(d)
	} else if taken < max {
		taken += b.TakeAvailable(max - taken)
	}

	return taken
}

type bitrateListener struct {
	net.Listener
}

func NewBitrateListener(l net.Listener) net.Listener {
	return &bitrateListener{l}
}

func (bl *bitrateListener) Accept() (net.Conn, error) {
	c, err := bl.Listener.Accept()
	if err != nil {
		return nil, err
	}

	wc, _ := c.(listeners.WrapConnEmbeddable)
	return &bitrateConn{
		WrapConnEmbeddable: wc,
		Conn:               c,
		limiter:            NewRateLimiter(0),
	}, err
}

// Bitrate Conn wrapper
type bitrateConn struct {
	listeners.WrapConnEmbeddable
	net.Conn
	limiter *RateLimiter
}

func (c *bitrateConn) Read(p []byte) (n int, err error) {
	lp := int64(len(p))
	s := c.limiter.takeRead(lp)
	if s != lp {
		p = p[:s]
	}
	return c.Conn.Read(p)
}

func (c *bitrateConn) Write(p []byte) (n int, err error) {
	var i int
	for lp := int64(len(p)); lp > 0 && err == nil; lp = int64(len(p)) {
		s := c.limiter.takeWrite(lp)
		i, err = c.Conn.Write(p[:s])
		p = p[i:]
		n += i
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
	// pro-user message always overrides the active flag
	if msgType == "throttle" {
		c.limiter = data.(*RateLimiter)
	}

	if c.WrapConnEmbeddable != nil {
		c.WrapConnEmbeddable.ControlMessage(msgType, data)
	}
}

func (c *bitrateConn) Wrapped() net.Conn {
	return c.Conn
}
