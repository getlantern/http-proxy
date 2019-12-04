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

type Instrument interface {
	WrapFilter(prefix string, f filters.Filter) filters.Filter
	WrapConnErrorHandler(prefix string, f func(conn net.Conn, err error)) func(conn net.Conn, err error)
	Blacklist(b bool)
	Mimic(m bool)
	Throttle(m bool, reason string)
	XBQHeaderSent()
	SuspectedProbing(reason string)
}

type NoInstrument struct {
}

func (i NoInstrument) WrapFilter(prefix string, f filters.Filter) filters.Filter { return f }
func (i NoInstrument) WrapConnErrorHandler(prefix string, f func(conn net.Conn, err error)) func(conn net.Conn, err error) {
	return f
}
func (i NoInstrument) Blacklist(b bool)               {}
func (i NoInstrument) Mimic(m bool)                   {}
func (i NoInstrument) Throttle(m bool, reason string) {}

func (i NoInstrument) XBQHeaderSent()                 {}
func (i NoInstrument) SuspectedProbing(reason string) {}

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

type instrumentedFilter struct {
	requests prometheus.Counter
	errors   prometheus.Counter
	duration prometheus.Observer
	filters.Filter
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

type promInstrument struct {
	commonLabels     prometheus.Labels
	commonLabelNames []string
	filters          map[string]*instrumentedFilter
	errorHandlers    map[string]func(conn net.Conn, err error)

	blacklist_checked, blacklisted, mimicry_checked, mimicked, xbqSent, throttling_checked prometheus.Counter

	throttled, notThrottled, suspectedProbing *prometheus.CounterVec
}

func NewPrometheus(c CommonLabels) Instrument {
	commonLabels := c.Labels()
	commonLabelNames := make([]string, len(commonLabels))
	i := 0
	for k := range commonLabels {
		commonLabelNames[i] = k
		i++
	}
	return &promInstrument{
		commonLabels:     commonLabels,
		commonLabelNames: commonLabelNames,
		filters:          make(map[string]*instrumentedFilter),
		errorHandlers:    make(map[string]func(conn net.Conn, err error)),
		blacklist_checked: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "blacklist_checked_requests_total",
		}, commonLabelNames).With(commonLabels),
		blacklisted: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "blacklist_blacklisted_requests_total",
		}, commonLabelNames).With(commonLabels),
		mimicry_checked: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "apache_mimicry_checked_total",
		}, commonLabelNames).With(commonLabels),
		mimicked: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "apache_mimicry_mimicked_total",
		}, commonLabelNames).With(commonLabels),

		xbqSent: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "xbq_header_sent_total",
		}, commonLabelNames).With(commonLabels),

		throttling_checked: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "device_throttling_checked_total",
		}, commonLabelNames).With(commonLabels),
		throttled: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "device_throttling_throttled_total",
		}, append(commonLabelNames, "reason")).MustCurryWith(commonLabels),
		notThrottled: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "device_throttling_not_throttled_total",
		}, append(commonLabelNames, "reason")).MustCurryWith(commonLabels),

		suspectedProbing: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "suspected_probing_total",
		}, append(commonLabelNames, "reason")).MustCurryWith(commonLabels),
	}
}

// Run runs the promInstrument exporter on the given address. The
// path is /metrics.
func (p *promInstrument) Run(addr string) error {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	server := http.Server{
		Addr:    addr,
		Handler: mux,
	}
	return server.ListenAndServe()
}

// WrapFilter wraps a filter to instrument the requests/errors/duration
// (so-called RED) of processed requests.
func (p *promInstrument) WrapFilter(prefix string, f filters.Filter) filters.Filter {
	wrapped := p.filters[prefix]
	if wrapped == nil {
		wrapped = &instrumentedFilter{
			promauto.NewCounterVec(prometheus.CounterOpts{
				Name: prefix + "_requests_total",
			}, p.commonLabelNames).With(p.commonLabels),
			promauto.NewCounterVec(prometheus.CounterOpts{
				Name: prefix + "_request_errors_total",
			}, p.commonLabelNames).With(p.commonLabels),
			promauto.NewHistogramVec(prometheus.HistogramOpts{
				Name:    prefix + "_request_duration_seconds",
				Buckets: []float64{0.001, 0.01, 0.1, 1},
			}, p.commonLabelNames).With(p.commonLabels),
			f}
		p.filters[prefix] = wrapped
	}
	return wrapped
}

// WrapConnErrorHandler wraps an error handler to instrument the error count.
func (p *promInstrument) WrapConnErrorHandler(prefix string, f func(conn net.Conn, err error)) func(conn net.Conn, err error) {
	h := p.errorHandlers[prefix]
	if h == nil {
		errors := promauto.NewCounterVec(prometheus.CounterOpts{
			Name: prefix + "_errors_total",
		}, p.commonLabelNames).With(p.commonLabels)
		consec_errors := promauto.NewCounterVec(prometheus.CounterOpts{
			Name: prefix + "_consec_per_client_ip_errors_total",
		}, p.commonLabelNames).With(p.commonLabels)
		if f == nil {
			f = func(conn net.Conn, err error) {}
		}
		var mu sync.Mutex
		var lastRemoteIP string
		h = func(conn net.Conn, err error) {
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
		p.errorHandlers[prefix] = h
	}
	return h
}

// Blacklist instruments the blacklist checking.
func (p *promInstrument) Blacklist(b bool) {
	p.blacklist_checked.Inc()
	if b {
		p.blacklisted.Inc()
	}
}

// Mimic instruments the Apache mimicry.
func (p *promInstrument) Mimic(m bool) {
	p.mimicry_checked.Inc()
	if m {
		p.mimicked.Inc()
	}
}

// Throttle instruments the device based throttling.
func (p *promInstrument) Throttle(m bool, reason string) {
	p.throttling_checked.Inc()
	if m {
		p.throttled.With(prometheus.Labels{"reason": reason}).Inc()
	} else {
		p.notThrottled.With(prometheus.Labels{"reason": reason}).Inc()
	}
}

// XBQHeaderSent counts the number of times XBQ header is sent along with the
// response.
func (p *promInstrument) XBQHeaderSent() {
	p.xbqSent.Inc()
}

// SuspectedProbing records the number of visits which looks like active
// probing.
func (p *promInstrument) SuspectedProbing(reason string) {
	p.suspectedProbing.With(prometheus.Labels{"reason": reason}).Inc()
}
