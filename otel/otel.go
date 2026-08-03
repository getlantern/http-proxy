package otel

import (
	"context"
	"net"
	"net/url"
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

// endpointURL parses a URL-form endpoint and appends the OTLP signal path
// (e.g. "v1/traces") since WithEndpointURL uses the URL path as-is. Callers
// can pass a base URL like https://collector/otlp and end up hitting
// https://collector/otlp/v1/traces. On parse failure, returns the endpoint
// unchanged.
func endpointURL(endpoint, signalPath string) string {
	u, err := url.Parse(endpoint)
	if err != nil {
		log.Errorf("failed to parse OTEL endpoint %q: %v", endpoint, err)
		return endpoint
	}
	return u.JoinPath(signalPath).String()
}

// isSecureHostPort reports whether a host:port endpoint targets port 443.
// Uses net.SplitHostPort so IPv6 literals and ports like 4430 that contain
// ":443" as a substring aren't misclassified.
func isSecureHostPort(endpoint string) bool {
	_, port, err := net.SplitHostPort(endpoint)
	if err != nil {
		return false
	}
	return port == "443"
}

func BuildTracerProvider(opts *Opts) (*sdktrace.TracerProvider, func()) {
	clientOpts := []otlptracehttp.Option{
		otlptracehttp.WithHeaders(opts.Headers),
	}
	if strings.Contains(opts.Endpoint, "://") {
		// URL form (e.g. https://collector:4318). WithEndpointURL handles
		// scheme and TLS derivation; append the signal path since it uses
		// the URL path as-is.
		clientOpts = append(clientOpts, otlptracehttp.WithEndpointURL(endpointURL(opts.Endpoint, "v1/traces")))
	} else {
		clientOpts = append(clientOpts, otlptracehttp.WithEndpoint(opts.Endpoint))
		if !isSecureHostPort(opts.Endpoint) {
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
		metricOpts = append(metricOpts, otlpmetrichttp.WithEndpointURL(endpointURL(opts.Endpoint, "v1/metrics")))
	} else {
		metricOpts = append(metricOpts, otlpmetrichttp.WithEndpoint(opts.Endpoint))
		if !isSecureHostPort(opts.Endpoint) {
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
