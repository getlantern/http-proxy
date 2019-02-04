package instrument

import (
	"net/http"
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
			Name: prefix + "_processed_requests_total",
		})).(prometheus.Counter),
		register(prometheus.NewCounter(prometheus.CounterOpts{
			Name: prefix + "_request_processing_errors_total",
		})).(prometheus.Counter),
		register(prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    prefix + "_request_processing_duration_seconds",
			Buckets: []float64{0.01, 0.1, 1},
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

// Blacklist instruments the blacklist checking.
func Blacklist() func(bool) {
	checked := register(prometheus.NewCounter(prometheus.CounterOpts{
		Name: "proxy_requests_blacklist_checked_total",
	})).(prometheus.Counter)
	blacklisted := register(prometheus.NewCounter(prometheus.CounterOpts{
		Name: "proxy_requests_blacklisted_total",
	})).(prometheus.Counter)

	return func(b bool) {
		checked.Inc()
		if b {
			blacklisted.Inc()
		}
	}
}

// Mimic instruments the Apache mimicry.
func Mimic() func(bool) {
	checked := register(prometheus.NewCounter(prometheus.CounterOpts{
		Name: "proxy_apache_mimicry_checked_total",
	})).(prometheus.Counter)
	mimicked := register(prometheus.NewCounter(prometheus.CounterOpts{
		Name: "proxy_apache_mimicked_total",
	})).(prometheus.Counter)

	return func(m bool) {
		checked.Inc()
		if m {
			mimicked.Inc()
		}
	}
}
