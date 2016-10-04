package ping

import (
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/getlantern/http-proxy-lantern/common"
	"github.com/getlantern/http-proxy/filters"
	"github.com/stretchr/testify/assert"
)

var (
	errNext = errors.New("nexterror")
)

func TestBypass(t *testing.T) {
	filter := New()
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "http://doesntmatter.domain", nil)
	n := &next{}
	err := filter.Apply(w, req, n.do)
	assert.Equal(t, errNext, err)
}

func TestInvalid(t *testing.T) {
	filter := New()
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "http://doesntmatter.domain", nil)
	req.Header.Set(common.PingHeader, "invalid")
	err := filter.Apply(w, req, filters.Stop)
	if assert.NoError(t, err) {
		assert.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
	}
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
	filter := New()
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "http://doesntmatter.domain", nil)
	req.Header.Set(common.PingHeader, size)
	err := filter.Apply(w, req, filters.Stop)
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
}

func (n *next) do() error {
	return errNext
}
