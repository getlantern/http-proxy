package otel

import (
	"context"
	"errors"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"

	otlpLog "go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	otelLog "go.opentelemetry.io/otel/log"
	otelLogSdk "go.opentelemetry.io/otel/sdk/log"
)

var logger otelLog.Logger

func InitLogger() error {
	service := "http-proxy-lantern"
	expLog, err := otlpLog.New(context.Background(),
		otlpLog.WithEndpoint("http://172.16.0.88:4317"),
		otlpLog.WithTLSClientConfig(tlsConfig),
	)
	if err != nil {
		return err
	}

	resourceAttributes := []attribute.KeyValue{
		semconv.ServiceNameKey.String(service),
	}

	r := resource.NewWithAttributes(semconv.SchemaURL, resourceAttributes...)

	provider := otelLogSdk.NewLoggerProvider(
		otelLogSdk.WithProcessor(otelLogSdk.NewBatchProcessor(expLog)),
		otelLogSdk.WithResource(r),
	)

	logger = provider.Logger(service)
	return nil
}

// For now I want to see if I can get logs to the otel collector
func Error(ctx context.Context, title string, err error, fields ...any) {
	if err == nil {
		err = errors.New(title)
	}

	var record otelLog.Record
	record.SetTimestamp(time.Now())
	record.SetBody(otelLog.StringValue(title))
	record.SetSeverity(otelLog.SeverityError)

	logger.Emit(ctx, record)
}
