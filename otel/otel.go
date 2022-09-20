package otel

import (
	"context"
	"sync"
	"time"

	"github.com/getlantern/golog"
	"github.com/getlantern/ops"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
)

var (
	log = golog.LoggerFor("otel")

	stopper   func()
	stopperMx sync.Mutex
)

func Configure(honeycombKey string, sampleRate int) {
	// Create HTTP client to talk to OTEL collector
	client := otlptracehttp.NewClient(
		otlptracehttp.WithEndpoint("api.honeycomb.io:443"),
		otlptracehttp.WithHeaders(map[string]string{
			"x-honeycomb-team": honeycombKey,
		}),
	)

	// Create an exporter that exports to the OTEL collector
	exporter, err := otlptrace.New(context.Background(), client)
	if err != nil {
		log.Errorf("Unable to initialize OpenTelemetry, will not report traces")
	} else {
		log.Debug("Will report traces to OpenTelemetry")
		// Create a TracerProvider that uses the above exporter
		resource :=
			resource.NewWithAttributes(
				semconv.SchemaURL,
				semconv.ServiceNameKey.String("http-proxy-lantern"),
			)
		tp := sdktrace.NewTracerProvider(
			sdktrace.WithBatcher(exporter),
			sdktrace.WithResource(resource),
			sdktrace.WithSampler(sdktrace.ParentBased(newDeterministicSampler(sampleRate))),
		)

		// Configure OTEL tracing to use the above TracerProvider
		otel.SetTracerProvider(tp)
		ops.EnableOpenTelemetry("http-proxy-lantern")

		stopperMx.Lock()
		if stopper != nil {
			// this means that we reconfigured after previously setting up a TracerProvider, shut the old one down
			go stopper()
		}
		stopper = func() {
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
			defer cancel()
			if err := tp.Shutdown(ctx); err != nil {
				log.Errorf("Error shutting down TracerProvider: %v", err)
			}
			if err := exporter.Shutdown(ctx); err != nil {
				log.Errorf("Error shutting down Exporter: %v", err)
			}
		}
		stopperMx.Unlock()
	}
}

func Stop() {
	stopperMx.Lock()
	defer stopperMx.Unlock()
	if stopper != nil {
		log.Debug("Stopping OpenTelemetry trace exporter")
		stopper()
		log.Debug("Stopped OpenTelemetry trace exporter")
	}
}
