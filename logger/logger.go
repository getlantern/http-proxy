package logger

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"path"
	"reflect"
	"runtime"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	"gopkg.in/ini.v1"

	"github.com/getlantern/golog"
	"github.com/mitchellh/mapstructure"
	"github.com/uptrace/opentelemetry-go-extra/otelutil"
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

// convertFields converts various types of fields into a slice of key-value pairs.
func convertFields(fields ...any) []any {
	var result []any
	for _, field := range fields {
		switch v := field.(type) {
		case []interface{}:
			for i := 0; i < len(v); i += 2 {
				if i+1 < len(v) {
					result = append(result, v[i], v[i+1])
				}
			}
		default:
			val := reflect.ValueOf(v)
			if val.Kind() == reflect.Map {
				for _, key := range val.MapKeys() {
					result = append(result, key.Interface(), val.MapIndex(key).Interface())
				}
			} else if val.Kind() == reflect.Struct {
				var mapResult map[string]interface{}
				err := mapstructure.Decode(v, &mapResult)
				if err == nil {
					result = append(result, v)
				}

			} else {
				result = append(result, v)
			}
		}
	}
	return result
}

func kvAttributes(vs []any) []otelLog.KeyValue {
	res := make([]otelLog.KeyValue, 0, len(vs)/2)

	var i int
	for i = 0; i+1 < len(vs); i += 2 {
		k, ok := vs[i].(string)
		if !ok {
			res = append(res, otelLog.String("logError", fmt.Sprintf("%+v is not a string key", vs[i])))
		}
		res = append(res, otelLog.KeyValue{Key: k, Value: otelutil.LogValue(vs[i+1])})
	}

	if i < len(vs) {
		res = append(res, otelLog.KeyValue{Key: "_EXTRA", Value: otelutil.LogValue(vs[i])})
	}

	return res
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

func (pl *ProxyLogger) writeLog(severity otelLog.Severity, message string, fields ...any) {
	if pl.otelLogger == nil {
		return
	}
	var record otelLog.Record
	record.SetTimestamp(time.Now())
	record.SetBody(otelLog.StringValue(message))
	record.SetSeverity(severity)
	record.SetSeverityText(severity.String())

	fields = convertFields(fields...)
	record.AddAttributes(kvAttributes(fields)...)

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
	pl.writeLog(otelLog.SeverityDebug, fmt.Sprintf("%v", message))
}

func (pl *ProxyLogger) Debugf(format string, args ...any) {
	if pl.stdLogger != nil {
		pl.stdLogger.Debugf(format, args...)
	}
	pl.writeLog(otelLog.SeverityDebug, format, args...)
}

func (pl *ProxyLogger) Fatal(message any) {
	if pl.stdLogger != nil {
		pl.stdLogger.Fatal(message)
	}

	var msg string

	switch v := message.(type) {
	case error:
		msg = v.Error()
	case fmt.Stringer:
		msg = v.String()
	case string:
		msg = v
	default:
		msg = "unknown error"
		return
	}

	pl.writeLog(otelLog.SeverityFatal, msg)
}

func (pl *ProxyLogger) Fatalf(format string, args ...any) {
	if pl.stdLogger != nil {
		pl.stdLogger.Fatalf(format, args...)
	}
	pl.writeLog(otelLog.SeverityFatal, format, args...)
}

func (pl *ProxyLogger) Trace(message any) {
	if pl.stdLogger != nil {
		pl.stdLogger.Trace(message)
	}
	var msg string

	switch v := message.(type) {
	case error:
		msg = v.Error()
	case fmt.Stringer:
		msg = v.String()
	case string:
		msg = v
	default:
		msg = "unknown error"
		return
	}

	pl.writeLog(otelLog.SeverityTrace, msg)
}

func (pl *ProxyLogger) Tracef(format string, args ...any) {
	if pl.stdLogger != nil {
		pl.stdLogger.Tracef(format, args...)
	}
	pl.writeLog(otelLog.SeverityTrace, format, args...)
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
