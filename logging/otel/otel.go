package otel

import (
	"context"
	"errors"
	"os"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/log"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
)

type Transport int

const (
	TransportHTTP Transport = iota
	TransportGRPC
)

type Config struct {
	ServiceName    string
	ServiceVersion string
	Environment    string
	Level          log.Severity
	FilePath       string
	Endpoint       string
	Transport      Transport
	Headers        map[string]string
	Insecure       bool
}

type OtelLogger struct {
	provider *sdklog.LoggerProvider
	logger   log.Logger
	level    log.Severity
	attrs    []log.KeyValue
}

func New(cfg Config) (*OtelLogger, error) {
	if cfg.FilePath == "" && cfg.Endpoint == "" {
		return nil, errors.New("at least one output (FilePath or Endpoint) is required")
	}

	if cfg.Level == log.SeverityUndefined {
		cfg.Level = log.SeverityInfo
	}

	res, err := buildResource(cfg)
	if err != nil {
		return nil, err
	}

	var opts []sdklog.LoggerProviderOption
	opts = append(opts, sdklog.WithResource(res))

	if cfg.FilePath != "" {
		fe, err := newFileExporter(cfg.FilePath)
		if err != nil {
			return nil, err
		}
		opts = append(opts, sdklog.WithProcessor(sdklog.NewSimpleProcessor(fe)))
	}

	if cfg.Endpoint != "" {
		exp, err := newOTLPExporter(cfg)
		if err != nil {
			return nil, err
		}
		opts = append(opts, sdklog.WithProcessor(sdklog.NewBatchProcessor(exp)))
	}

	provider := sdklog.NewLoggerProvider(opts...)
	logger := provider.Logger("komodo-otel-logger")

	return &OtelLogger{
		provider: provider,
		logger:   logger,
		level:    cfg.Level,
	}, nil
}

func (otel *OtelLogger) Trace(ctx context.Context, msg string, attrs ...log.KeyValue) {
	otel.emit(ctx, log.SeverityTrace, msg, attrs)
}

func (otel *OtelLogger) Debug(ctx context.Context, msg string, attrs ...log.KeyValue) {
	otel.emit(ctx, log.SeverityDebug, msg, attrs)
}

func (otel *OtelLogger) Info(ctx context.Context, msg string, attrs ...log.KeyValue) {
	otel.emit(ctx, log.SeverityInfo, msg, attrs)
}

func (otel *OtelLogger) Warn(ctx context.Context, msg string, attrs ...log.KeyValue) {
	otel.emit(ctx, log.SeverityWarn, msg, attrs)
}

func (otel *OtelLogger) Error(ctx context.Context, msg string, attrs ...log.KeyValue) {
	otel.emit(ctx, log.SeverityError, msg, attrs)
}

func (otel *OtelLogger) Fatal(ctx context.Context, msg string, attrs ...log.KeyValue) {
	otel.emit(ctx, log.SeverityFatal, msg, attrs)
}

func (otel *OtelLogger) Publish(ctx context.Context, rec log.Record) {
	otel.logger.Emit(ctx, rec)
}

func (otel *OtelLogger) Enabled(ctx context.Context, severity log.Severity) bool {
	return severity >= otel.level
}

func (otel *OtelLogger) WithAttributes(attrs ...log.KeyValue) *OtelLogger {
	merged := make([]log.KeyValue, len(otel.attrs)+len(attrs))
	copy(merged, otel.attrs)
	copy(merged[len(otel.attrs):], attrs)
	return &OtelLogger{
		provider: otel.provider,
		logger:   otel.logger,
		level:    otel.level,
		attrs:    merged,
	}
}

func (otel *OtelLogger) Shutdown(ctx context.Context) error {
	return otel.provider.Shutdown(ctx)
}

func (otel *OtelLogger) ForceFlush(ctx context.Context) error {
	return otel.provider.ForceFlush(ctx)
}

func (otel *OtelLogger) emit(ctx context.Context, sev log.Severity, msg string, attrs []log.KeyValue) {
	if sev < otel.level {
		return
	}
	var rec log.Record
	rec.SetTimestamp(time.Now())
	rec.SetSeverity(sev)
	rec.SetSeverityText(sev.String())
	rec.SetBody(log.StringValue(msg))

	if len(otel.attrs) > 0 {
		rec.AddAttributes(otel.attrs...)
	}
	if len(attrs) > 0 {
		rec.AddAttributes(attrs...)
	}

	otel.logger.Emit(ctx, rec)
}

func String(key, value string) log.KeyValue          { return log.String(key, value) }
func Int(key string, value int) log.KeyValue         { return log.Int(key, value) }
func Int64(key string, value int64) log.KeyValue     { return log.Int64(key, value) }
func Float64(key string, value float64) log.KeyValue { return log.Float64(key, value) }
func Bool(key string, value bool) log.KeyValue       { return log.Bool(key, value) }
func Bytes(key string, value []byte) log.KeyValue    { return log.Bytes(key, value) }

func buildResource(cfg Config) (*resource.Resource, error) {
	svcName := cfg.ServiceName
	if svcName == "" {
		svcName = os.Getenv("OTEL_SERVICE_NAME")
	}
	if svcName == "" {
		svcName = "unknown_service"
	}

	svcVersion := cfg.ServiceVersion
	if svcVersion == "" {
		svcVersion = os.Getenv("OTEL_SERVICE_VERSION")
	}

	env := cfg.Environment
	if env == "" {
		env = os.Getenv("OTEL_DEPLOYMENT_ENVIRONMENT")
	}

	attrs := []attribute.KeyValue{
		attribute.String("service.name", svcName),
	}
	if svcVersion != "" {
		attrs = append(attrs, attribute.String("service.version", svcVersion))
	}
	if env != "" {
		attrs = append(attrs, attribute.String("deployment.environment", env))
	}

	return resource.NewSchemaless(attrs...), nil
}
