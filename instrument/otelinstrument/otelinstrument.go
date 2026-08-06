// Provides an OpenTelemetry version of our instrumentation.
// TODO: when we're ready to switch off prometheus and once the OTEL metrics
// SDK is stable, consider removing the Intrument interface and just
// using the OTEL metrics API at the point where the relevant metrics are being
// gathered.
package otelinstrument

import (
	"context"
	"errors"
	"flag"
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
	SessionGoodput                                           metric.Float64Histogram
	DistinctClients1m, DistinctClients10m, DistinctClients1h *distinct.SlidingWindowDistinctCount
	distinctClients                                          metric.Int64ObservableGauge
)

// GoodputBucketBoundaries are the explicit bucket boundaries, in bytes/s, for
// the proxy.session.goodput histogram: a half-decade log scale from 1 B/s to
// 10 MB/s. Session goodput spans ~6 decades — idle keepalive sessions sit
// below 100 B/s while real page loads run 100 kB/s–10 MB/s — and the SDK
// default boundaries top out at 10,000 B/s, which censored everything faster
// than 10 kB/s into the final bucket (IR's daily p90 read as exactly 10000).
// A log scale gives every decade the same relative resolution, so
// p50/p90/p99 interpolate to meaningful values across the whole range.
//
// Exactly 15 boundaries, matching the SDK default's count: SigNoz stores one
// sample per bucket per series per export interval, so keeping the count
// identical keeps the metric's ingest cost identical (this family is ~20% of
// metric ingest; see lantern-cloud #3069).
//
// lantern-box emits the same metric from its tracker/metrics package and MUST
// use identical boundaries — SigNoz merges the two streams, and quantiles
// over mixed bucket layouts are garbage.
var GoodputBucketBoundaries = []float64{
	1, 3, 10, 30, 100, 300,
	1_000, 3_000, 10_000, 30_000, 100_000, 300_000,
	1_000_000, 3_000_000, 10_000_000,
}

// Note - we don't use package-level init() because we want to defer initialization of
// OTEL metrics until after we've configured the global meter provider.
func Initialize() error {
	var err error
	initOnce.Do(func() {
		err = initialize()
	})
	return err
}

// ResetForTest re-runs initialization against the current global meter provider.
// initialize() is guarded by a sync.Once, so once the process has initialized
// the instruments they stay bound to whichever meter provider was global at the
// time — a meter provider swapped in later (e.g. a test's in-memory reader) is
// ignored. Tests call this after installing their own provider so the package's
// instruments record into that provider's reader. It must not be used outside
// tests.
//
// It mutates package-global initialization state (initOnce, meter, and the
// instrument handles) without synchronization, so it is only safe for
// sequential tests; do not call it from tests that run with t.Parallel(). It
// refuses to run outside a `go test` binary (detected via the test.v flag) so
// a stray production call can't re-run initialization and race live metric use.
func ResetForTest() error {
	if flag.Lookup("test.v") == nil {
		return errors.New("otelinstrument: ResetForTest may only be called from a test binary")
	}
	initOnce = sync.Once{}
	return Initialize()
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
	if Connections, err = meter.Int64Counter("proxy.connections"); err != nil {
		return err
	}
	// Per-session goodput (received bytes per second of connection lifetime),
	// recorded once at connection close for any session that moved received
	// bytes over a positive lifetime (see instrument.SessionGoodput for why
	// there is no byte floor). "received" is the client→proxy direction, tagged
	// network.io.direction="receive". Sliceable by track × geo.country.iso_code
	// (both point attrs) so the bandit experiment evaluator can compare a
	// challenger track's median goodput against the incumbent's per market;
	// cloud.region stays a resource attr. Unit "bytes/s" follows proxy.io's
	// "bytes" spelling for consistency within this package's metrics.
	if SessionGoodput, err = meter.Float64Histogram("proxy.session.goodput",
		metric.WithUnit("bytes/s"),
		metric.WithDescription("Per-session goodput: received (client->proxy) bytes per second of connection lifetime"),
		metric.WithExplicitBucketBoundaries(GoodputBucketBoundaries...)); err != nil {
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
