package ping

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
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
	assert.EqualValues(t, 0.927, int(s.mathisThroughput()))
	s.plr.Set(0)
	assert.EqualValues(t, 45.421, int(s.mathisThroughput()))
}

func TestBypass(t *testing.T) {
	filter, err := New(nil)
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

func TestGoodPlain(t *testing.T) {
	doTestGood(t, false)
}

func TestGoodGZ(t *testing.T) {
	doTestGood(t, true)
}

func doTestGood(t *testing.T, acceptGZ bool) {
	filter, err := New()
	if !assert.NoError(t, err) {
		return
	}
	defer func() {
		log.Debug("Closing")
		filter.(*pingMiddleware).pinger.Close()
	}()

	// Give the filter some time to pick up new timings
	time.Sleep(15 * time.Second)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "http://doesntmatter.domain", nil)
	// Note - the actual value of this header doesn't matter, it just needs to be
	// set to something.
	req.Header.Set(common.PingHeader, "summary")
	if acceptGZ {
		// Add multiple accept-encoding headers to make sure it doesn't confuse
		// server
		req.Header.Add("Accept-Encoding", "compress")
		req.Header.Add("Accept-Encoding", "gzip")
		req.Header.Add("Accept-Encoding", "deflate")
	}
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
	body, err := ioutil.ReadAll(resp.Body)
	if !assert.NoError(t, err) {
		return
	}
	if acceptGZ {
		if !assert.Equal(t, "gzip", resp.Header.Get("Content-Encoding")) {
			return
		}
		// Decompress
		r, err := gzip.NewReader(bytes.NewReader(body))
		if !assert.NoError(t, err) {
			return
		}
		body, err = ioutil.ReadAll(r)
		if !assert.NoError(t, err) {
			return
		}
	}

	summary := make(map[string]map[string]interface{})
	err = json.Unmarshal(body, &summary)
	if !assert.NoError(t, err) {
		return
	}

	if !assert.True(t, len(summary) > 0) {
		return
	}
	gvid := summary["googlevideo.com"]
	if !assert.NotNil(t, gvid) {
		return
	}
	tput := gvid["tput"]
	if !assert.NotNil(t, tput) {
		return
	}
	assert.NotEqual(t, defaultEMAStats.mathisThroughput(), tput, "Should have gotten non-default throughput")
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
