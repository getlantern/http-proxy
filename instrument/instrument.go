package instrument

import (
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/getlantern/proxy/filters"
)

// Start starts the Prometheus exporter on the given address. The
// path is /metrics.
func Start(addr string) error {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	server := http.Server{
		Addr:    addr,
		Handler: mux,
	}
	return server.ListenAndServe()
}

func register(c prometheus.Collector) prometheus.Collector {
	if err := prometheus.Register(c); err != nil {
		if are, ok := err.(prometheus.AlreadyRegisteredError); ok {
			// A counter for that metric has been registered before.
			// Use the old counter from now on. It's to avoid panic in tests,
			// which register the collectors more than once.
			return are.ExistingCollector
		} else {
			panic(err)
		}
	}
	return c
}

type instrumentedFilter struct {
	requests prometheus.Counter
	errors   prometheus.Counter
	duration prometheus.Histogram
	filters.Filter
}

// WrapFilter wraps a filter to instrument the requests/errors/duration
// (so-called RED) of processed requests.
func WrapFilter(prefix string, f filters.Filter) filters.Filter {
	return &instrumentedFilter{
		register(prometheus.NewCounter(prometheus.CounterOpts{
			Name: prefix + "_requests_total",
		})).(prometheus.Counter),
		register(prometheus.NewCounter(prometheus.CounterOpts{
			Name: prefix + "_request_errors_total",
		})).(prometheus.Counter),
		register(prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    prefix + "_request_duration_seconds",
			Buckets: []float64{0.001, 0.01, 0.1, 1},
		})).(prometheus.Histogram),
		f}
}

func (f *instrumentedFilter) Apply(ctx filters.Context, req *http.Request, next filters.Next) (*http.Response, filters.Context, error) {
	start := time.Now()
	res, ctx, err := f.Filter.Apply(ctx, req, next)
	f.requests.Inc()
	if err != nil {
		f.errors.Inc()
	}
	f.duration.Observe(time.Since(start).Seconds())
	return res, ctx, err
}

// WrapConnErrorHandler wraps an error handler to instrument the error count.
func WrapConnErrorHandler(prefix string, f func(conn net.Conn, err error)) func(conn net.Conn, err error) {
	errors := register(prometheus.NewCounter(prometheus.CounterOpts{
		Name: prefix + "_errors_total",
	})).(prometheus.Counter)
	consec_errors := register(prometheus.NewCounter(prometheus.CounterOpts{
		Name: prefix + "_consec_per_client_ip_errors_total",
	})).(prometheus.Counter)
	if f == nil {
		f = func(conn net.Conn, err error) {}
	}
	var mu sync.Mutex
	var lastRemoteIP string
	return func(conn net.Conn, err error) {
		errors.Inc()
		addr := conn.RemoteAddr()
		if addr == nil {
			panic("nil RemoteAddr")
		}
		host, _, err := net.SplitHostPort(addr.String())
		if err != nil {
			panic("Unexpected RemoteAddr " + addr.String())
		}
		mu.Lock()
		if lastRemoteIP != host {
			lastRemoteIP = host
			mu.Unlock()
			consec_errors.Inc()
		} else {
			mu.Unlock()
		}
		f(conn, err)
	}
}

var (
	blacklist_checked = register(prometheus.NewCounter(prometheus.CounterOpts{
		Name: "blacklist_checked_requests_total",
	})).(prometheus.Counter)
	blacklisted = register(prometheus.NewCounter(prometheus.CounterOpts{
		Name: "blacklist_blacklisted_requests_total",
	})).(prometheus.Counter)
	mimicry_checked = register(prometheus.NewCounter(prometheus.CounterOpts{
		Name: "apache_mimicry_checked_total",
	})).(prometheus.Counter)
	mimicked = register(prometheus.NewCounter(prometheus.CounterOpts{
		Name: "apache_mimicry_mimicked_total",
	})).(prometheus.Counter)

	throttling_checked = register(prometheus.NewCounter(prometheus.CounterOpts{
		Name: "device_throttling_checked_total",
	})).(prometheus.Counter)
	throttled = register(prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "device_throttling_throttled_total",
	}, []string{"reason"})).(*prometheus.CounterVec)

	notThrottled = register(prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "device_throttling_not_throttled_total",
	}, []string{"reason"})).(*prometheus.CounterVec)
	xbqSent = register(prometheus.NewCounter(prometheus.CounterOpts{
		Name: "device_throttling_xbq_header_sent_total",
	})).(prometheus.Counter)

	suspectedProbing = register(prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "suspected_probing_total",
	}, []string{"reason"})).(*prometheus.CounterVec)
)

// Blacklist instruments the blacklist checking.
func Blacklist(b bool) {
	blacklist_checked.Inc()
	if b {
		blacklisted.Inc()
	}
}

// Mimic instruments the Apache mimicry.
func Mimic(m bool) {
	mimicry_checked.Inc()
	if m {
		mimicked.Inc()
	}
}

// Throttle instruments the device based throttling.
func Throttle(m bool, reason string) {
	throttling_checked.Inc()
	if m {
		throttled.With(prometheus.Labels{"reason": reason}).Inc()
	} else {
		notThrottled.With(prometheus.Labels{"reason": reason}).Inc()
	}
}

// XBQHeaderSent counts the number of times XBQ header is sent along with the
// response.
func XBQHeaderSent() {
	xbqSent.Inc()
}

// SuspectedProbing records the number of visits which looks like active
// probing.
func SuspectedProbing(reason string) {
	suspectedProbing.With(prometheus.Labels{"reason": reason}).Inc()
}
