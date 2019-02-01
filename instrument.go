package proxy

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/getlantern/proxy/filters"
)

type instrumentedFilter struct {
	requests prometheus.Counter
	errors   prometheus.Counter
	duration prometheus.Histogram
	filters.Filter
}

// Instrumented wraps a filter to instrument the requests/errors/duration
// (so-called RED) of processed requests.
func Instrumented(prefix string, f filters.Filter) filters.Filter {
	return &instrumentedFilter{
		promauto.NewCounter(prometheus.CounterOpts{
			Name: prefix + "_processed_requests_total",
		}),
		promauto.NewCounter(prometheus.CounterOpts{
			Name: prefix + "_request_processing_errors_total",
		}),
		promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    prefix + "_request_processing_duration_seconds",
			Buckets: []float64{0.01, 0.1, 1},
		}),
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

// StartPromExporter starts the Prometheus exporter on the given address. The
// path is /metrics.
func StartPromExporter(addr string) error {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	server := http.Server{
		Addr:    addr,
		Handler: mux,
	}
	return server.ListenAndServe()
}
