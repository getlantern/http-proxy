// package ping provides a ping-like service that gives insight into the
// performance of this proxy.
package ping

import (
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/getlantern/golog"

	"github.com/getlantern/http-proxy/filters"

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

func (pm *pingMiddleware) Apply(w http.ResponseWriter, req *http.Request, next filters.Next) error {
	log.Trace("In ping")
	pingSize := req.Header.Get(common.PingHeader)
	pingURL := req.Header.Get(common.PingURLHeader)
	if pingSize == "" && pingURL == "" {
		log.Trace("Bypassing ping")
		return next()
	}
	log.Trace("Processing ping")

	if pingURL != "" {
		return pm.urlPing(w, pingURL)
	}

	var size int
	switch pingSize {
	case "small":
		size = 1
	case "medium":
		size = 5
	case "large":
		size = 10
	default:
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Invalid ping size %v\n", pingSize)
		return filters.Stop()
	}

	return pm.cannedPing(w, size)
}

func (pm *pingMiddleware) cannedPing(w http.ResponseWriter, size int) error {
	w.WriteHeader(200)
	// Simulate latency by sleeping
	time.Sleep(time.Duration(size) * time.Second)
	// Flush to the client to make sure we're getting a comprehensive timing
	return filters.Stop()
}
