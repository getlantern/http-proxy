// package ping provides a ping-like service that gives insight into the
// performance of this proxy.
package ping

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/getlantern/golog"
	"github.com/getlantern/proxy/filters"

	"github.com/getlantern/http-proxy-lantern/common"
)

var (
	log = golog.LoggerFor("http-proxy-lantern.ping")

	letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

	// Data is 1 KB of random data
	data []byte
)

func init() {
	rand.Seed(time.Now().UnixNano())

	data = []byte(randStringRunes(1024))
	data[1023] = '\n'
}

// randStringRunes generates a random string of the given length.
// Taken from http://stackoverflow.com/questions/22892120/how-to-generate-a-random-string-of-a-fixed-length-in-golang.
func randStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

// pingMiddleware intercepts ping requests and returns some random data
type pingMiddleware struct {
	timingExpiration time.Duration
	urlTimings       map[string]*urlTiming
	urlTimingsMx     sync.RWMutex
}

func New(timingExpiration time.Duration) filters.Filter {
	if timingExpiration <= 0 {
		timingExpiration = defaultTimingExpiration
	}
	pm := &pingMiddleware{
		timingExpiration: timingExpiration,
		urlTimings:       make(map[string]*urlTiming),
	}
	go pm.cleanupExpiredTimings()
	return pm
}

func (pm *pingMiddleware) Apply(ctx context.Context, req *http.Request, next filters.Next) (*http.Response, context.Context, error) {
	log.Trace("In ping")
	pingSize := req.Header.Get(common.PingHeader)
	pingURL := req.Header.Get(common.PingURLHeader)
	isPingURL := req.Host == "ping-chained-server"
	if pingSize == "" && pingURL == "" && !isPingURL {
		log.Trace("Bypassing ping")
		return next(ctx, req)
	}
	log.Trace("Processing ping")

	if pingURL != "" {
		return pm.urlPing(ctx, req, pingURL)
	}

	var size int
	switch pingSize {
	case "small":
		size = 1
	case "medium":
		size = 100
	case "large":
		size = 10000
	default:
		var parseErr error
		size, parseErr = strconv.Atoi(pingSize)
		if parseErr != nil && isPingURL {
			size, parseErr = strconv.Atoi(req.URL.RawQuery)
		}
		if parseErr != nil {
			return filters.Fail(ctx, req, http.StatusBadRequest, fmt.Errorf("Invalid ping size %v\n", pingSize))
		}
	}

	return pm.cannedPing(ctx, req, size)
}

func (pm *pingMiddleware) cannedPing(ctx context.Context, req *http.Request, size int) (*http.Response, context.Context, error) {
	return filters.ShortCircuit(ctx, req, &http.Response{
		StatusCode: http.StatusOK,
		Body:       &randReader{size * len(data)},
	})
}

type randReader struct {
	remain int
}

func (r *randReader) Read(b []byte) (int, error) {
	n := 0
	for len(b) > 0 && r.remain > 0 {
		toCopy := len(b)
		if toCopy > len(data) {
			toCopy = len(data)
		}
		if toCopy > r.remain {
			toCopy = r.remain
		}
		copy(b, data[:toCopy])
		b = b[toCopy:]
		r.remain -= toCopy
		n += toCopy
	}
	var err error
	if r.remain == 0 {
		err = io.EOF
	}
	return n, err
}

func (r *randReader) Close() error {
	return nil
}
