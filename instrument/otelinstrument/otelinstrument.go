// Provides an OpenTelemetry version of our Prometheus-based instrumentation.
// TODO: when we're ready to switch off prometheus and once the OTEL metrics
// SDK is stable, consider removing the Intrument interface and just
// using the OTEL metrics API at the point where the relevant metrics are being
// gathered.
package otelinstrument

import (
	"net/http"
	"sync"
	"time"

	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/global"
	"go.opentelemetry.io/otel/metric/instrument"

	"github.com/getlantern/proxy/v2/filters"
)

var (
	initOnce         sync.Once
	meter            metric.Meter
	Blacklist        instrument.Int64Counter
	ProxyIO          instrument.Int64Counter
	QuicPackets      instrument.Int64Counter
	Mimicked         instrument.Int64Counter
	MultipathFrames  instrument.Int64Counter
	MultipathIO      instrument.Int64Counter
	XBQ              instrument.Int64Counter
	Throttling       instrument.Int64Counter
	SuspectedProbing instrument.Int64Counter
	VersionCheck     instrument.Int64Counter
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
	meter = global.MeterProvider().Meter("")
	var err error
	if ProxyIO, err = meter.Int64Counter("proxy.io", instrument.WithUnit("bytes")); err != nil {
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
	if MultipathIO, err = meter.Int64Counter("proxy.multipath.io", instrument.WithUnit("bytes")); err != nil {
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
	if VersionCheck, err = meter.Int64Counter("proxy.version.checked"); err != nil {
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
	requests instrument.Int64Counter
	errors   instrument.Int64Counter
	duration instrument.Float64Histogram
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

func ConnErrorHandlerCounter(prefix string) (instrument.Int64Counter, error) {
	return meter.Int64Counter(prefix + "_errors_total")
}

func ConnConsecErrorHandlerCounter(prefix string) (instrument.Int64Counter, error) {
	return meter.Int64Counter(prefix + "_consec_per_client_ip_errors_total")
}
