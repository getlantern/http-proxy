package otel

import (
	"context"
	"strings"
	"time"

	sdkotel "go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"

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
	Endpoint             string
	Headers              map[string]string
	SampleRate           int
	ExternalIP           string
	ProxyName            string
	Track                string
	Provider             string
	DC                   string
	FrontendProvider     string
	FrontendDC           string
	ProxyProtocol        string
	Addr                 string
	IsPro                bool
	IncludeProxyIdentity bool
}

func (opts *Opts) buildResource() *resource.Resource {
	attributes := []attribute.KeyValue{
		semconv.ServiceNameKey.String("http-proxy-lantern"),
		attribute.String("protocol", opts.ProxyProtocol),
		attribute.Bool("pro", opts.IsPro),
	}
	// Disable reporting proxy port for Datadog cost reasons
	// parts := strings.Split(opts.Addr, ":")
	// if len(parts) == 2 {
	// 	_port := parts[1]
	// 	port, err := strconv.Atoi(_port)
	// 	if err == nil {
	// 		log.Debugf("will report with proxy.port %d", port)
	// 		attributes = append(attributes, attribute.Int("proxy.port", port))
	// 	} else {
	// 		log.Errorf("Unable to parse proxy.port %v: %v", _port, err)
	// 	}
	// } else {
	// 	log.Errorf("Unable to split proxy address %v into two pieces", opts.Addr)
	// }
	if opts.Track != "" {
		attributes = append(attributes, attribute.String("track", opts.Track))
	}
	// Disable reporting proxy IP for Datadog cost reasons
	// if opts.ExternalIP != "" {
	// 	log.Debugf("Will report with proxy.ip: %v", opts.ExternalIP)
	// 	attributes = append(attributes, attribute.String("proxy.ip", opts.ExternalIP))
	// }
	if opts.ProxyName != "" {
		log.Debugf("Will report with proxy.name %v on provider %v in dc %v", opts.ProxyName, opts.Provider, opts.DC)
		// Disable reporting proxy name for Datadog cost reasons
		// attributes = append(attributes, attribute.String("proxy.name", opts.ProxyName))
		attributes = append(attributes, attribute.String("provider", opts.Provider))
		attributes = append(attributes, attribute.String("dc", opts.DC))
	}
	if opts.FrontendProvider != "" {
		log.Debugf("Will report frontend provider %v in dc %v", opts.FrontendProvider, opts.FrontendDC)
		attributes = append(attributes, attribute.String("frontend.provider", opts.FrontendProvider))
		attributes = append(attributes, attribute.String("frontend.dc", opts.FrontendDC))
	}
	attributes = append(
		attributes,
		attribute.Bool("legacy", strings.HasPrefix(opts.ProxyName, "fp-")),
	)
	return resource.NewWithAttributes(semconv.SchemaURL, attributes...)
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
		sdktrace.WithSampler(sdktrace.ParentBased(newDeterministicSampler(opts.SampleRate))),
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
