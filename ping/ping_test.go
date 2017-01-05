package ping

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/getlantern/ema"
	"github.com/getlantern/http-proxy-lantern/common"
	"github.com/stretchr/testify/assert"
)

var (
	errNext = errors.New("nexterror")
)

func TestMathisThroughput(t *testing.T) {
	s := &emaStats{
		rtt: ema.NewDuration(115*time.Millisecond, 0.5),
		plr: ema.New(1.2, 0.5),
	}
	assert.Equal(t, 927, int(s.mathisThroughput()))
	s.plr.Set(0)
	assert.Equal(t, 45421, int(s.mathisThroughput()))
}

func TestBypass(t *testing.T) {
	filter, err := New()
	if !assert.NoError(t, err) {
		return
	}
	defer func() {
		filter.(*pingMiddleware).pinger.Close()
	}()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "http://doesntmatter.domain", nil)
	n := &next{}
	err = filter.Apply(w, req, n.do)
	assert.True(t, n.wasCalled())
	assert.Equal(t, errNext, err)
}

func TestGood(t *testing.T) {
	filter, err := New()
	if !assert.NoError(t, err) {
		return
	}
	defer func() {
		filter.(*pingMiddleware).pinger.Close()
	}()

	// Give the filter some time to pick up new timings
	time.Sleep(1 * time.Minute)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "http://doesntmatter.domain", nil)
	req.Header.Set(common.PingOriginHeader, "FaceBook.com")
	n := &next{}
	err = filter.Apply(w, req, n.do)
	assert.False(t, n.wasCalled())
	if !assert.NoError(t, err) {
		return
	}
	resp := w.Result()
	if !assert.Equal(t, http.StatusOK, resp.StatusCode) {
		return
	}
	defer resp.Body.Close()
	throughput, err := strconv.ParseFloat(resp.Header.Get(common.PingThroughputHeader), 64)
	if !assert.NoError(t, err) {
		return
	}
	assert.True(t, throughput > 0)
	assert.NotEqual(t, defaultEMAStats.mathisThroughput(), throughput, "Should have gotten non-default throughput")
}

type next struct {
	called bool
	mx     sync.Mutex
}

func (n *next) do() error {
	n.mx.Lock()
	n.called = true
	n.mx.Unlock()
	return errNext
}

func (n *next) wasCalled() bool {
	n.mx.Lock()
	defer n.mx.Unlock()
	return n.called
}
