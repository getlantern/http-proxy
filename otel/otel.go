package otel

import (
	"context"
	"sync"
	"time"

	"github.com/getlantern/golog"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/global"
	"go.opentelemetry.io/otel/metric/instrument/syncint64"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
)

const (
	reportInterval = 15 * time.Second
	serviceName    = "http-proxy-lantern"
)

var (
	log = golog.LoggerFor("otel")

	meter    metric.Meter
	counters = make(map[string]syncint64.Counter)

	stopper func()
	mx      sync.Mutex
)

type Opts struct {
	HoneycombKey  string
	SampleRate    int
	ExternalIP    string
	ProxyName     string
	DC            string
	ProxyProtocol string
	IsPro         bool
}

func Configure(opts *Opts) {
	// Create HTTP client to talk to OTEL collector
	exporter, err := otlpmetrichttp.New(context.Background(),
		otlpmetrichttp.WithEndpoint("api.honeycomb.io:443"),
		otlpmetrichttp.WithHeaders(map[string]string{
			"x-honeycomb-team":    opts.HoneycombKey,
			"x-honeycomb-dataset": serviceName,
		}),
	)
	if err != nil {
		log.Errorf("Unable to create OpenTelemetry metrics client, will not report metrics: %v", err)
		return
	}

	log.Debug("Will report metrics to OpenTelemetry")

	attributes := []attribute.KeyValue{
		semconv.ServiceNameKey.String(serviceName),
		attribute.String("proxy_protocol", opts.ProxyProtocol),
		attribute.Bool("pro", opts.IsPro),
	}
	if opts.ExternalIP != "" {
		log.Debugf("Will report with external_ip: %v", opts.ExternalIP)
		attributes = append(attributes, attribute.String("external_ip", opts.ExternalIP))
	}
	// Only set proxy name if it follows our naming convention
	if opts.ProxyName != "" {
		log.Debugf("Will report with proxy_name %v in dc %v", opts.ProxyName, opts.DC)
		attributes = append(attributes, attribute.String("proxy_name", opts.ProxyName))
		attributes = append(attributes, attribute.String("dc", opts.DC))
	}
	resource := resource.NewWithAttributes(semconv.SchemaURL, attributes...)
	m := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(resource),
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exporter, sdkmetric.WithInterval(reportInterval))),
	)
	global.SetMeterProvider(m)

	mx.Lock()
	if stopper != nil {
		// this means that we reconfigured after previously setting up a MeterProvider,
		// reset counters and shut the old one down
		counters = make(map[string]syncint64.Counter)
		go stopper()
	}
	meter = global.Meter("github.com/getlantern/http-proxy-lantern")
	stopper = func() {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
		defer cancel()
		if err := m.Shutdown(ctx); err != nil {
			log.Errorf("Error stopping metrics meter provider: %v", err)
		}
		if err := exporter.Shutdown(ctx); err != nil {
			log.Errorf("Error shutting down metrics exporter: %v", err)
		}
	}
	mx.Unlock()
}

func CounterAdd(ctx context.Context, name string, value int, attributes ...attribute.KeyValue) {
	mx.Lock()
	defer mx.Unlock()

	if meter == nil {
		// meter not initialized, ignore
		return
	}

	counter, found := counters[name]
	if !found {
		var err error
		counter, err = meter.SyncInt64().Counter(name)
		if err != nil {
			// unable to initialize counter, skip
			return
		}
		counters[name] = counter
	}
	counter.Add(ctx, int64(value), attributes...)
}

func Stop() {
	mx.Lock()
	defer mx.Unlock()
	if stopper != nil {
		log.Debug("Stopping OpenTelemetry metrics exporter")
		stopper()
		log.Debug("Stopped OpenTelemetry metrics exporter")
	}
}
