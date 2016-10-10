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
	"github.com/getlantern/http-proxy-lantern/metrics"
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
	smallResponseTime  metrics.MovingAverage
	mediumResponseTime metrics.MovingAverage
	largeResponseTime  metrics.MovingAverage
	timingExpiration   time.Duration
	urlTimings         map[string]*urlTiming
	urlTimingsMx       sync.RWMutex
}

func New(timingExpiration time.Duration) filters.Filter {
	if timingExpiration <= 0 {
		timingExpiration = defaultTimingExpiration
	}
	pm := &pingMiddleware{
		smallResponseTime:  metrics.NewMovingAverage(),
		mediumResponseTime: metrics.NewMovingAverage(),
		largeResponseTime:  metrics.NewMovingAverage(),
		timingExpiration:   timingExpiration,
		urlTimings:         make(map[string]*urlTiming),
	}
	go pm.logTimings()
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
	var ma metrics.MovingAverage
	switch pingSize {
	case "small":
		size = 1
		ma = pm.smallResponseTime
	case "medium":
		size = 100
		ma = pm.mediumResponseTime
	case "large":
		size = 10000
		ma = pm.largeResponseTime
	default:
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Invalid ping size %v\n", pingSize)
		return filters.Stop()
	}

	return pm.cannedPing(w, ma, size)
}

func (pm *pingMiddleware) cannedPing(w http.ResponseWriter, ma metrics.MovingAverage, size int) error {
	start := time.Now()
	w.WriteHeader(200)
	for i := 0; i < size; i++ {
		w.Write(data)
	}
	// Flush to the client to make sure we're getting a comprehensive timing
	w.(http.Flusher).Flush()
	delta := time.Now().Sub(start)
	ma.Update(delta.Nanoseconds() / 1000)

	return filters.Stop()
}

func (pm *pingMiddleware) logTimings() {
	for {
		time.Sleep(1 * time.Minute)
		now := time.Now()
		msg := fmt.Sprintf(`**** Average Ping Response Times in µs, moving average (1 min, 5 min, 15 min) ****
%v Small      (1 KB) - %v
%v Medium   (100 KB) - %v
%v Large (10,000 KB) - %v
`, now, pm.smallResponseTime,
			now, pm.mediumResponseTime,
			now, pm.largeResponseTime)
		log.Debug(msg)
	}
}
