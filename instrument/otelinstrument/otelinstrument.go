// Provides an OpenTelemetry version of our instrumentation.
// TODO: when we're ready to switch off prometheus and once the OTEL metrics
// SDK is stable, consider removing the Intrument interface and just
// using the OTEL metrics API at the point where the relevant metrics are being
// gathered.
package otelinstrument

import (
	"context"
	"net/http"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/getlantern/http-proxy-lantern/v2/instrument/distinct"
	"github.com/getlantern/proxy/v3/filters"
)

var (
	initOnce                                                 sync.Once
	meter                                                    metric.Meter
	Blacklist                                                metric.Int64Counter
	ProxyIO                                                  metric.Int64Counter
	QuicPackets                                              metric.Int64Counter
	Mimicked                                                 metric.Int64Counter
	MultipathFrames                                          metric.Int64Counter
	MultipathIO                                              metric.Int64Counter
	XBQ                                                      metric.Int64Counter
	Throttling                                               metric.Int64Counter
	SuspectedProbing                                         metric.Int64Counter
	Connections                                              metric.Int64Counter
	DistinctClients1m, DistinctClients10m, DistinctClients1h *distinct.SlidingWindowDistinctCount
	distinctClients                                          metric.Int64ObservableGauge
)

// Note - we don't use package-level init() because we want to defer initialization of
// OTEL metrics until after we've configured the global meter provider.
func Initialize() error {
	var err error
	initOnce.Do(func() {
		err = initialize()
	})
	return err
}

func initialize() error {
	meter = otel.GetMeterProvider().Meter("")
	var err error
	if ProxyIO, err = meter.Int64Counter("proxy.io", metric.WithUnit("bytes")); err != nil {
		return err
	}
	if QuicPackets, err = meter.Int64Counter("proxy.quic.packets"); err != nil {
		return err
	}
	if Mimicked, err = meter.Int64Counter("proxy.apache.mimicked"); err != nil {
		return err
	}
	if MultipathFrames, err = meter.Int64Counter("proxy.multipath.frames"); err != nil {
		return err
	}
	if MultipathIO, err = meter.Int64Counter("proxy.multipath.io", metric.WithUnit("bytes")); err != nil {
		return err
	}
	if XBQ, err = meter.Int64Counter("proxy.xbq.headers"); err != nil {
		return err
	}
	if Throttling, err = meter.Int64Counter("proxy.clients.throttling"); err != nil {
		return err
	}
	if Blacklist, err = meter.Int64Counter("proxy.clients.blacklist"); err != nil {
		return err
	}
	if SuspectedProbing, err = meter.Int64Counter("proxy.probing.suspected"); err != nil {
		return err
	}
	if Connections, err = meter.Int64Counter("proxy.connections", metric.WithUnit("connections")); err != nil {
		return err
	}

	DistinctClients1m = distinct.NewSlidingWindowDistinctCount(time.Minute, time.Second)
	DistinctClients10m = distinct.NewSlidingWindowDistinctCount(10*time.Minute, 10*time.Second)
	DistinctClients1h = distinct.NewSlidingWindowDistinctCount(time.Hour, time.Minute)

	if distinctClients, err = meter.Int64ObservableGauge(
		"proxy.clients.active",
		metric.WithInt64Callback(func(ctx context.Context, io metric.Int64Observer) error {
			io.Observe(int64(DistinctClients1m.Cardinality()), metric.WithAttributes(attribute.String("window", "1m")))
			io.Observe(int64(DistinctClients10m.Cardinality()), metric.WithAttributes(attribute.String("window", "10m")))
			io.Observe(int64(DistinctClients1h.Cardinality()), metric.WithAttributes(attribute.String("window", "1h")))
			return nil
		})); err != nil {
		return err
	}
	return nil
}

func WrapFilter(prefix string, f filters.Filter) (filters.Filter, error) {
	result := &instrumentedFilter{
		Filter: f,
	}
	var err error
	if result.requests, err = meter.Int64Counter(prefix + "_requests_total"); err != nil {
		return nil, err
	}
	if result.errors, err = meter.Int64Counter(prefix + "_request_errors_total"); err != nil {
		return nil, err
	}
	if result.duration, err = meter.Float64Histogram(prefix + "_request_duration_seconds"); err != nil {
		return nil, err
	}
	return result, nil
}

type instrumentedFilter struct {
	filters.Filter
	requests metric.Int64Counter
	errors   metric.Int64Counter
	duration metric.Float64Histogram
}

func (f *instrumentedFilter) Apply(cs *filters.ConnectionState, req *http.Request, next filters.Next) (*http.Response, *filters.ConnectionState, error) {
	start := time.Now()
	res, cs, err := f.Filter.Apply(cs, req, next)
	f.requests.Add(req.Context(), 1)
	if err != nil {
		f.errors.Add(req.Context(), 1)
	}
	f.duration.Record(req.Context(), time.Since(start).Seconds())

	return res, cs, err
}

func ConnErrorHandlerCounter(prefix string) (metric.Int64Counter, error) {
	return meter.Int64Counter(prefix + "_errors_total")
}

func ConnConsecErrorHandlerCounter(prefix string) (metric.Int64Counter, error) {
	return meter.Int64Counter(prefix + "_consec_per_client_ip_errors_total")
}
