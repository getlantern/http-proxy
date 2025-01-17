package otel

import (
	"context"
	"crypto/tls"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"

	otlpLog "go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	otelLog "go.opentelemetry.io/otel/log"
	otelLogSdk "go.opentelemetry.io/otel/sdk/log"
)

var logger otelLog.Logger

func init() {
	InitLogger()
}

func InitLogger() error {
	service := "http-proxy-lantern"
	expLog, err := otlpLog.New(context.Background(),
		otlpLog.WithEndpoint("http://172.16.0.88:4317"),
		otlpLog.WithTLSClientConfig(&tls.Config{InsecureSkipVerify: true}),
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

func Debug(ctx context.Context, title string) {
	var record otelLog.Record
	record.SetTimestamp(time.Now())
	record.SetBody(otelLog.StringValue(title))
	record.SetSeverity(otelLog.SeverityDebug)

	logger.Emit(ctx, record)
}
