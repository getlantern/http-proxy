package ping

import (
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/getlantern/http-proxy-lantern/common"
	"github.com/getlantern/http-proxy/filters"
	"github.com/stretchr/testify/assert"
)

var (
	errNext = errors.New("nexterror")
)

func TestBypass(t *testing.T) {
	filter := New(0)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "http://doesntmatter.domain", nil)
	n := &next{}
	err := filter.Apply(w, req, n.do)
	assert.True(t, n.wasCalled())
	assert.Equal(t, errNext, err)
}

func TestInvalid(t *testing.T) {
	filter := New(0)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "http://doesntmatter.domain", nil)
	req.Header.Set(common.PingHeader, "invalid")
	n := &next{}
	err := filter.Apply(w, req, n.do)
	assert.False(t, n.wasCalled())
	if assert.NoError(t, err) {
		assert.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
	}
}

func TestGoodURL(t *testing.T) {
	timingExpiration := 5 * time.Second
	goodURL := "https://www.google.com/humans.txt"
	badURL := "https://www.google.com/unknown.txt"
	filter := New(timingExpiration)
	statusCode, badTS := doTestURL(t, filter, badURL)
	if !assert.Equal(t, http.StatusNotFound, statusCode) {
		return
	}

	statusCode, firstTS := doTestURL(t, filter, goodURL)
	if !assert.Equal(t, http.StatusOK, statusCode) {
		return
	}
	assert.NotEqual(t, badTS, firstTS, "Bad timing should not have been cached")

	statusCode, secondTS := doTestURL(t, filter, goodURL)
	if !assert.Equal(t, http.StatusOK, statusCode) {
		return
	}
	assert.Equal(t, firstTS, secondTS, "Should have used cached timing on 2nd request")

	time.Sleep(timingExpiration * 2)
	statusCode, thirdTS := doTestURL(t, filter, goodURL)
	if !assert.Equal(t, http.StatusOK, statusCode) {
		return
	}
	assert.NotEqual(t, secondTS, thirdTS, "Should have gotten new timing on 3rd request")

	time.Sleep(timingExpiration * 2)
	pingFilter := filter.(*pingMiddleware)
	pingFilter.urlTimingsMx.RLock()
	defer pingFilter.urlTimingsMx.RUnlock()
	assert.Empty(t, pingFilter.urlTimings)
}

func doTestURL(t *testing.T, filter filters.Filter, url string) (statusCode int, ts string) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "http://doesntmatter.domain", nil)
	req.Header.Set(common.PingURLHeader, url)
	n := &next{}
	err := filter.Apply(w, req, n.do)
	assert.False(t, n.wasCalled())
	if assert.NoError(t, err) {
		resp := w.Result()
		statusCode = resp.StatusCode
		if resp.StatusCode == http.StatusOK {
			n, _ := io.Copy(ioutil.Discard, w.Result().Body)
			assert.EqualValues(t, 0, n)
			ts = resp.Header.Get(common.PingTSHeader)
			assert.NotEmpty(t, ts)
		}
	}

	return
}

func TestSmall(t *testing.T) {
	testSize(t, "small", 1)
}

func TestMedium(t *testing.T) {
	testSize(t, "medium", 100)
}

func TestLarge(t *testing.T) {
	testSize(t, "large", 10000)
}

func testSize(t *testing.T, size string, mult int) {
	filter := New(0)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "http://doesntmatter.domain", nil)
	req.Header.Set(common.PingHeader, size)
	n := &next{}
	err := filter.Apply(w, req, n.do)
	assert.False(t, n.wasCalled())
	if assert.NoError(t, err) {
		resp := w.Result()
		if assert.Equal(t, http.StatusOK, resp.StatusCode) {
			n, _ := io.Copy(ioutil.Discard, w.Result().Body)
			assert.EqualValues(t, mult*len(data), n)
		}
	}
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
