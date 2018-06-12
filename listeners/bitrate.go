package listeners

import (
	"net"
	"net/http"
	"sync"

	"github.com/juju/ratelimit"

	"github.com/getlantern/http-proxy/listeners"
)

type RateLimiter struct {
	r    *ratelimit.Bucket
	w    *ratelimit.Bucket
	rate int64
	mx   sync.RWMutex
}

func NewRateLimiter(rate int64) *RateLimiter {
	l := &RateLimiter{}
	l.SetRate(rate)
	return l
}

func (l *RateLimiter) SetRate(rate int64) {
	l.mx.Lock()
	if l.rate != rate {
		l.rate = rate
		if rate > 0 {
			l.r = ratelimit.NewBucketWithRate(float64(rate), rate)
			l.w = ratelimit.NewBucketWithRate(float64(rate), rate)
		} else {
			l.r = nil
			l.w = nil
		}
	}
	l.mx.Unlock()
}

func (l *RateLimiter) getReadBucket() (b *ratelimit.Bucket) {
	l.mx.RLock()
	b = l.r
	l.mx.RUnlock()
	return
}

func (l *RateLimiter) getWriteBucket() (b *ratelimit.Bucket) {
	l.mx.RLock()
	b = l.w
	l.mx.RUnlock()
	return
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
	bucket := c.limiter.getReadBucket()
	if bucket != nil {
		s := bucket.TakeAvailable(int64(len(p)))
		if s == 0 {
			bucket.Wait(1)
			s = 1
		}
		p = p[:s]
	}

	return c.Conn.Read(p)
}

func (c *bitrateConn) Write(p []byte) (n int, err error) {
	bucket := c.limiter.getWriteBucket()
	if bucket == nil {
		return c.Conn.Write(p)
	}

	var i int
	for len(p) > 0 && err == nil {
		s := bucket.TakeAvailable(int64(len(p)))
		if s == 0 {
			bucket.Wait(1)
			s = 1
		}
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
