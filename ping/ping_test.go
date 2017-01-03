package ping

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/getlantern/http-proxy-lantern/common"
	"github.com/stretchr/testify/assert"
)

var (
	errNext = errors.New("nexterror")
)

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
	time.Sleep(5 * time.Minute)

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
	rtt, err := time.ParseDuration(resp.Header.Get(common.PingRTTHeader))
	if !assert.NoError(t, err) {
		return
	}
	plr, err := strconv.ParseFloat(resp.Header.Get(common.PingPLRHeader), 64)
	if !assert.NoError(t, err) {
		return
	}
	log.Debug(rtt)
	log.Debug(plr)
	assert.True(t, rtt > 0)
	assert.True(t, plr >= 0)
	assert.NotEqual(t, defaultStats.rtt, rtt, "Should have gotten non-default rtt")
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
