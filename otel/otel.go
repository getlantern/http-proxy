package otel

import (
	"context"
	"time"

	lanternsc "github.com/getlantern/semconv"
	sdkotel "go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	otelsc "go.opentelemetry.io/otel/semconv/v1.37.0"

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
		otelsc.ServiceNameKey.String("http-proxy-lantern"),
		lanternsc.ProxyProtocolKey.String(opts.ProxyProtocol),
		lanternsc.ClientIsProKey.Bool(opts.IsPro),
		attribute.Bool("legacy", opts.Legacy),
	}
	if opts.Track != "" {
		attrs = append(attrs,
			lanternsc.ProxyTrackKey.String(opts.Track))
	}
	if opts.ProxyName != "" {
		attrs = append(attrs,
			lanternsc.ProxyNameKey.String(opts.ProxyName))
	}
	if opts.Provider != "" {
		attrs = append(attrs,
			lanternsc.ProxyProviderKey.String(opts.Provider))
	}
	if opts.DC != "" {
		attrs = append(attrs, attribute.String("dc", opts.DC))
	}
	if opts.FrontendProvider != "" {
		attrs = append(attrs,
			lanternsc.ProxyFrontendProviderKey.String(opts.FrontendProvider),
			attribute.String("frontend.dc", opts.FrontendDC),
		)
	}
	log.Debugf("Resource attributes: %v", attrs)
	return resource.NewWithAttributes(otelsc.SchemaURL, attrs...)
}

func BuildTracerProvider(opts *Opts) (*sdktrace.TracerProvider, func()) {
	// Create HTTP client to talk to OTEL collector
	client := otlptracehttp.NewClient(
		otlptracehttp.WithEndpoint(opts.Endpoint),
		otlptracehttp.WithHeaders(opts.Headers),
	)

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
	exp, err := otlpmetrichttp.New(context.Background(),
		otlpmetrichttp.WithEndpoint(opts.Endpoint),
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
	)
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
