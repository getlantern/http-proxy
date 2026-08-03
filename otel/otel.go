package otel

import (
	"context"
	"strings"
	"time"

	semconv "github.com/getlantern/semconv"
	sdkotel "go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"github.com/getlantern/golog"
)

const (
	batchTimeout = 1 * time.Minute
	maxQueueSize = 10000
)

var (
	log = golog.LoggerFor("otel")
)

type Opts struct {
	Endpoint         string
	Headers          map[string]string
	ProxyName        string
	Track            string
	Provider         string
	DC               string
	FrontendProvider string
	FrontendDC       string
	ProxyProtocol    string
	Addr             string
	IsPro            bool
	Legacy           bool
}

func (opts *Opts) buildResource() *resource.Resource {
	attrs := []attribute.KeyValue{
		semconv.ServiceNameKey.String("http-proxy-lantern"),
		semconv.ProxyProtocolKey.String(opts.ProxyProtocol),
		semconv.ClientIsProKey.Bool(opts.IsPro),
		attribute.Bool("legacy", opts.Legacy),
	}
	if opts.Track != "" {
		attrs = append(attrs,
			semconv.ProxyTrackKey.String(opts.Track))
	}
	if opts.ProxyName != "" {
		attrs = append(attrs,
			semconv.ProxyNameKey.String(opts.ProxyName))
	}
	if opts.Provider != "" {
		attrs = append(attrs,
			semconv.ProxyProviderKey.String(opts.Provider))
	}
	if opts.DC != "" {
		attrs = append(attrs, attribute.String("dc", opts.DC))
	}
	if opts.FrontendProvider != "" {
		attrs = append(attrs,
			semconv.ProxyFrontendProviderKey.String(opts.FrontendProvider),
			attribute.String("frontend.dc", opts.FrontendDC),
		)
	}
	log.Debugf("Resource attributes: %v", attrs)

	// Merge with the SDK default resource so OTEL_RESOURCE_ATTRIBUTES,
	// OTEL_SERVICE_NAME, and host detection (host.name from OS) all flow in.
	// resource.Merge(a, b) prefers b on duplicate keys. Put env-derived attrs
	// LAST so deployment identity (service.name, host.name, etc — supplied via
	// OTEL_RESOURCE_ATTRIBUTES by the launcher) wins over the code's built-in
	// fallbacks. The runtime attrs above (proxy.protocol, is_pro, track, ...)
	// don't overlap with any env-supplied key today, so this only affects
	// identity fields whose whole purpose is deployment-time override.
	base, err := resource.New(context.Background(),
		resource.WithFromEnv(),
		resource.WithHost(),
	)
	if err != nil {
		log.Errorf("resource.New failed, falling back to explicit-only: %v", err)
		return resource.NewWithAttributes(semconv.SchemaURL, attrs...)
	}
	explicit := resource.NewWithAttributes(semconv.SchemaURL, attrs...)
	merged, err := resource.Merge(explicit, base)
	if err != nil {
		log.Errorf("resource.Merge failed, falling back to explicit-only: %v", err)
		return explicit
	}
	return merged
}

func BuildTracerProvider(opts *Opts) (*sdktrace.TracerProvider, func()) {
	clientOpts := []otlptracehttp.Option{
		otlptracehttp.WithHeaders(opts.Headers),
	}
	if strings.Contains(opts.Endpoint, "://") {
		// URL-form (e.g. http://localhost:4318). WithEndpointURL handles the
		// scheme, host:port, and TLS/insecure derivation.
		clientOpts = append(clientOpts, otlptracehttp.WithEndpointURL(opts.Endpoint))
	} else {
		clientOpts = append(clientOpts, otlptracehttp.WithEndpoint(opts.Endpoint))
		// host:port form: keep the historical heuristic that anything not on
		// :443 is plaintext.
		if !strings.Contains(opts.Endpoint, ":443") {
			log.Debugf("Using insecure connection for OTEL endpoint %v", opts.Endpoint)
			clientOpts = append(clientOpts, otlptracehttp.WithInsecure())
		}
	}

	client := otlptracehttp.NewClient(clientOpts...)

	// Create an exporter that exports to the OTEL collector
	exporter, err := otlptrace.New(context.Background(), client)
	if err != nil {
		log.Errorf("Unable to initialize OpenTelemetry, will not report traces to %v", opts.Endpoint)
		return nil, func() {}
	}
	log.Debugf("Will report traces to OpenTelemetry at %v", opts.Endpoint)

	// Create a TracerProvider that uses the above exporter
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(
			exporter,
			sdktrace.WithBatchTimeout(batchTimeout),
			sdktrace.WithMaxQueueSize(maxQueueSize),
			sdktrace.WithBlocking(), // it's okay to use blocking mode right now because we're just submitting bandwidth data in a goroutine that doesn't block real work
		),
		sdktrace.WithResource(opts.buildResource()),
	)

	stop := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
		defer cancel()
		if err := tp.Shutdown(ctx); err != nil {
			log.Errorf("Error shutting down TracerProvider: %v", err)
		}
		if err := exporter.Shutdown(ctx); err != nil {
			log.Errorf("Error shutting down Exporter: %v", err)
		}
	}

	return tp, stop
}

func InitGlobalMeterProvider(opts *Opts) (func(), error) {
	metricOpts := []otlpmetrichttp.Option{
		otlpmetrichttp.WithHeaders(opts.Headers),
		otlpmetrichttp.WithTemporalitySelector(func(kind sdkmetric.InstrumentKind) metricdata.Temporality {
			switch kind {
			case
				sdkmetric.InstrumentKindCounter,
				sdkmetric.InstrumentKindUpDownCounter,
				sdkmetric.InstrumentKindObservableCounter,
				sdkmetric.InstrumentKindObservableUpDownCounter:
				return metricdata.DeltaTemporality
			default:
				return metricdata.CumulativeTemporality
			}
		}),
	}
	if strings.Contains(opts.Endpoint, "://") {
		metricOpts = append(metricOpts, otlpmetrichttp.WithEndpointURL(opts.Endpoint))
	} else {
		metricOpts = append(metricOpts, otlpmetrichttp.WithEndpoint(opts.Endpoint))
		if !strings.Contains(opts.Endpoint, ":443") {
			log.Debugf("Using insecure connection for OTEL metrics endpoint %v", opts.Endpoint)
			metricOpts = append(metricOpts, otlpmetrichttp.WithInsecure())
		}
	}

	exp, err := otlpmetrichttp.New(context.Background(), metricOpts...)
	if err != nil {
		return nil, err
	}

	// Create a new meter provider
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exp)),
		sdkmetric.WithResource(opts.buildResource()),
	)

	// Set the meter provider as global
	sdkotel.SetMeterProvider(mp)

	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		err := mp.Shutdown(ctx)
		if err != nil {
			log.Errorf("error shutting down meter provider: %v", err)
		}
	}, nil
}
