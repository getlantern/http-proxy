package otel

import (
	"context"
	"os"
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

	// Merge in the SDK's env and host detectors so OTEL_RESOURCE_ATTRIBUTES,
	// OTEL_SERVICE_NAME, and host.name (from OS) all flow in.
	// resource.Merge(a, b) prefers b on duplicate keys. WithFromEnv runs LAST
	// so OTEL_RESOURCE_ATTRIBUTES (launcher-supplied deployment identity) wins
	// over the OS host detector, and the merge below places base after the
	// runtime attrs so env identity also wins over the code's built-in
	// fallbacks. The runtime attrs above (proxy.protocol, is_pro, track, ...)
	// don't overlap with any env-supplied key today.
	//
	// explicit is schemaless so the merge succeeds regardless of the SDK
	// detector semconv version (SDK v1.35.0 detectors emit v1.26.0 while
	// getlantern/semconv currently mirrors v1.34.0).
	base, err := resource.New(context.Background(),
		resource.WithHost(),
		resource.WithFromEnv(),
	)
	if err != nil {
		log.Errorf("resource.New returned error: %v", err)
	}
	explicit := resource.NewSchemaless(attrs...)
	if base == nil {
		return explicit
	}
	merged, mergeErr := resource.Merge(explicit, base)
	if mergeErr != nil {
		log.Errorf("resource.Merge failed, falling back to explicit-only: %v", mergeErr)
		return explicit
	}
	return merged
}

// logExporterEndpoint reports the OTLP endpoint the SDK will use for the
// given signal ("traces" or "metrics"), or logs a loud error if the env
// isn't configured. This is the only visibility the operator gets that
// telemetry wiring landed, since the SDK silently falls back to
// localhost:4318 when nothing is set.
func logExporterEndpoint(signal string) {
	perSignal := "OTEL_EXPORTER_OTLP_" + map[string]string{
		"traces":  "TRACES_ENDPOINT",
		"metrics": "METRICS_ENDPOINT",
	}[signal]
	if ep := os.Getenv(perSignal); ep != "" {
		log.Debugf("OTel %s exporter using %s=%s", signal, perSignal, ep)
		return
	}
	if ep := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"); ep != "" {
		log.Debugf("OTel %s exporter using OTEL_EXPORTER_OTLP_ENDPOINT=%s", signal, ep)
		return
	}
	log.Errorf("OTel %s exporter: neither %s nor OTEL_EXPORTER_OTLP_ENDPOINT is set; "+
		"SDK will default to localhost:4318 and exports will silently fail",
		signal, perSignal)
}

// BuildTracerProvider constructs a TracerProvider that exports over OTLP/HTTP.
// The exporter's endpoint, scheme, TLS, headers, and other transport settings
// come from the standard OTEL_EXPORTER_OTLP_* env vars (see the OpenTelemetry
// SDK env spec).
func BuildTracerProvider(opts *Opts) (*sdktrace.TracerProvider, func()) {
	logExporterEndpoint("traces")
	client := otlptracehttp.NewClient()

	exporter, err := otlptrace.New(context.Background(), client)
	if err != nil {
		log.Errorf("Unable to initialize OpenTelemetry tracer: %v", err)
		return nil, func() {}
	}

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

// InitGlobalMeterProvider sets a global MeterProvider that exports over
// OTLP/HTTP. Endpoint, scheme, TLS, headers, and other transport settings
// are read from the standard OTEL_EXPORTER_OTLP_* env vars.
func InitGlobalMeterProvider(opts *Opts) (func(), error) {
	logExporterEndpoint("metrics")
	exp, err := otlpmetrichttp.New(context.Background(),
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

	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exp)),
		sdkmetric.WithResource(opts.buildResource()),
	)
	sdkotel.SetMeterProvider(mp)

	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := mp.Shutdown(ctx); err != nil {
			log.Errorf("error shutting down meter provider: %v", err)
		}
	}, nil
}
