package instrument

import (
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/getlantern/proxy/filters"
)

type CommonLabels struct {
	Protocol              string
	SupportTLSResumption  bool
	RequireTLSResumption  bool
	MissingTicketReaction string
}

func (c *CommonLabels) Labels() prometheus.Labels {
	return map[string]string{
		"protocol":                c.Protocol,
		"support_tls_resumption":  strconv.FormatBool(c.SupportTLSResumption),
		"require_tls_resumption":  strconv.FormatBool(c.RequireTLSResumption),
		"missing_ticket_reaction": c.MissingTicketReaction,
	}
}

var (
	commonLabelNames = []string{
		"protocol",
		"support_tls_resumption",
		"require_tls_resumption",
		"missing_ticket_reaction",
	}

	commonLabels CommonLabels

	blacklist_checked, blacklisted, mimicry_checked, mimicked, xbqSent, throttling_checked prometheus.Counter

	throttled, notThrottled, suspectedProbing *prometheus.CounterVec
)

func initCollectors() {
	blacklist_checked = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "blacklist_checked_requests_total",
	}, commonLabelNames).With(commonLabels.Labels())
	blacklisted = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "blacklist_blacklisted_requests_total",
	}, commonLabelNames).With(commonLabels.Labels())
	mimicry_checked = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "apache_mimicry_checked_total",
	}, commonLabelNames).With(commonLabels.Labels())
	mimicked = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "apache_mimicry_mimicked_total",
	}, commonLabelNames).With(commonLabels.Labels())

	xbqSent = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "device_throttling_xbq_header_sent_total",
	}, commonLabelNames).With(commonLabels.Labels())

	throttling_checked = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "device_throttling_checked_total",
	}, commonLabelNames).With(commonLabels.Labels())
	throttled = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "device_throttling_throttled_total",
	}, append(commonLabelNames, "reason")).MustCurryWith(commonLabels.Labels())

	notThrottled = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "device_throttling_not_throttled_total",
	}, append(commonLabelNames, "reason")).MustCurryWith(commonLabels.Labels())

	suspectedProbing = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "suspected_probing_total",
	}, append(commonLabelNames, "reason")).MustCurryWith(commonLabels.Labels())

}

// Start starts the Prometheus exporter on the given address. The
// path is /metrics.
func Start(addr string, c CommonLabels) error {
	commonLabels = c
	initCollectors()
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
		register(prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: prefix + "_requests_total",
		}, commonLabelNames).With(commonLabels.Labels())).(prometheus.Counter),
		register(prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: prefix + "_request_errors_total",
		}, commonLabelNames).With(commonLabels.Labels())).(prometheus.Counter),
		register(prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    prefix + "_request_duration_seconds",
			Buckets: []float64{0.001, 0.01, 0.1, 1},
		}, commonLabelNames).With(commonLabels.Labels()).(prometheus.Histogram)).(prometheus.Histogram),
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
	errors := register(prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: prefix + "_errors_total",
	}, commonLabelNames).With(commonLabels.Labels())).(prometheus.Counter)
	consec_errors := register(prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: prefix + "_consec_per_client_ip_errors_total",
	}, commonLabelNames).With(commonLabels.Labels())).(prometheus.Counter)
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
