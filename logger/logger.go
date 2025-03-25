package logger

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"path"
	"runtime"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	"gopkg.in/ini.v1"

	"github.com/getlantern/golog"
	otlpLog "go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	otelLog "go.opentelemetry.io/otel/log"
	otelLogSdk "go.opentelemetry.io/otel/sdk/log"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

const (
	otelEndpoint    = "172.16.0.88:4318"
	otelServiceName = "http-proxy-lantern"
)

// wraps both an otel logger and std logger
type ProxyLogger struct {
	stdLogger  golog.Logger
	otelLogger otelLog.Logger
}

type Opts struct {
	ProviderMachine string `ini:"provider"`
	TrackName       string `ini:"track"`
	RouteName       string `ini:"proxyname"`
}

func (o Opts) attrKV() []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String("provider", o.ProviderMachine),
		attribute.String("track", o.TrackName),
		attribute.String("route", o.RouteName),
	}
}

func (pl *ProxyLogger) SetStdLogger(logger golog.Logger) *ProxyLogger {
	nL := &ProxyLogger{
		stdLogger: logger,
	}

	otelLogger, _ := BuildOtelLogger(BuildOtelOptsFromINI())
	nL.otelLogger = otelLogger
	return nL
}

func BuildOtelOptsFromINI() Opts {
	cfg, err := ini.Load("/home/lantern/config.ini")
	if err != nil {
		return Opts{}
	}

	var opts Opts
	err = cfg.MapTo(&opts)
	if err != nil {
		return Opts{}
	}

	return opts
}

func BuildOtelLogger(opts Opts) (otelLog.Logger, error) {
	expLog, err := otlpLog.New(context.Background(),
		otlpLog.WithEndpoint(otelEndpoint),
		otlpLog.WithInsecure(), // the endpoint is on the lo interface, so this "might" be safe
	)
	if err != nil {
		return nil, err
	}

	resourceAttributes := []attribute.KeyValue{semconv.ServiceNameKey.String(otelServiceName)}
	resourceAttributes = append(resourceAttributes, opts.attrKV()...)

	r := resource.NewWithAttributes(semconv.SchemaURL, resourceAttributes...)
	provider := otelLogSdk.NewLoggerProvider(
		otelLogSdk.WithProcessor(otelLogSdk.NewBatchProcessor(expLog)),
		otelLogSdk.WithResource(r),
	)

	return provider.Logger(otelServiceName), nil
}

func InitLogger(stdLoggerPrefix string) *ProxyLogger {
	goLog := golog.LoggerFor(stdLoggerPrefix)
	p := &ProxyLogger{
		stdLogger: goLog,
	}

	oLogger, err := BuildOtelLogger(BuildOtelOptsFromINI())
	if err != nil {
		return p
	}

	p.otelLogger = oLogger
	return p
}

func (pl *ProxyLogger) writeLog(severity otelLog.Severity, message any) {
	if pl.otelLogger == nil {
		return
	}
	var record otelLog.Record
	record.SetTimestamp(time.Now())
	record.SetBody(otelLog.StringValue(fmt.Sprintf("%v", message)))
	record.SetSeverity(severity)
	record.SetSeverityText(severity.String())

	if pc, file, line, ok := runtime.Caller(2); ok {
		fn := ""
		if function := runtime.FuncForPC(pc); function != nil {
			fn = function.Name()
		}
		record.AddAttributes(otelLog.String("file", path.Base(file)), otelLog.Int64("line", int64(line)), otelLog.String("function", fn))
	}

	pl.otelLogger.Emit(context.Background(), record)
}

func (pl *ProxyLogger) Debug(message any) {
	if pl.stdLogger != nil {
		pl.stdLogger.Debug(message)
	}
	pl.writeLog(otelLog.SeverityDebug, message)
}

func (pl *ProxyLogger) Debugf(format string, args ...any) {
	if pl.stdLogger != nil {
		pl.stdLogger.Debugf(format, args...)
	}
	pl.writeLog(otelLog.SeverityDebug, fmt.Sprintf(format, args...))
}

func (pl *ProxyLogger) Fatal(message any) {
	if pl.stdLogger != nil {
		pl.stdLogger.Fatal(message)
	}
	pl.writeLog(otelLog.SeverityFatal, message)
}

func (pl *ProxyLogger) Fatalf(format string, args ...any) {
	if pl.stdLogger != nil {
		pl.stdLogger.Fatalf(format, args...)
	}
	pl.writeLog(otelLog.SeverityFatal, fmt.Sprintf(format, args...))
}

func (pl *ProxyLogger) Trace(message any) {
	if pl.stdLogger != nil {
		pl.stdLogger.Trace(message)
	}

	pl.writeLog(otelLog.SeverityTrace, message)
}

func (pl *ProxyLogger) Tracef(format string, args ...any) {
	if pl.stdLogger != nil {
		pl.stdLogger.Tracef(format, args...)
	}
	pl.writeLog(otelLog.SeverityTrace, fmt.Sprintf(format, args...))
}

func (pl *ProxyLogger) Error(message any) error {
	var err error
	var msg string

	switch v := message.(type) {
	case error:
		msg = v.Error()
		err = v
	case fmt.Stringer:
		msg = v.String()
		err = errors.New(msg)
	case string:
		msg = v
		err = errors.New(v)
	default:
		msg = "unknown error"
		err = errors.New(msg)
	}

	if pl.stdLogger != nil {
		pl.stdLogger.Error(msg)
	}

	pl.writeLog(otelLog.SeverityError, msg)
	return err
}

func (pl *ProxyLogger) Errorf(format string, args ...any) error {
	var e error
	if pl.stdLogger != nil {
		e = pl.stdLogger.Errorf(format, args...)
	}

	err := fmt.Errorf(format, args...)
	msg := err.Error()

	pl.writeLog(otelLog.SeverityError, msg)
	return e
}

func (pl *ProxyLogger) TraceOut() io.Writer        { return pl.stdLogger.TraceOut() }
func (pl *ProxyLogger) IsTraceEnabled() bool       { return pl.stdLogger.IsTraceEnabled() }
func (pl *ProxyLogger) AsDebugLogger() *log.Logger { return pl.stdLogger.AsDebugLogger() }
func (pl *ProxyLogger) AsStdLogger() *log.Logger   { return pl.stdLogger.AsStdLogger() }
func (pl *ProxyLogger) AsErrorLogger() *log.Logger { return pl.stdLogger.AsStdLogger() }
