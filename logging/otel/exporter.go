package otel

import (
	"context"

	sdklog "go.opentelemetry.io/otel/sdk/log"

	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
)

func newOTLPExporter(cfg Config) (sdklog.Exporter, error) {
	ctx := context.Background()
	switch cfg.Transport {
	case TransportGRPC:
		return newGRPCExporter(ctx, cfg)
	default:
		return newHTTPExporter(ctx, cfg)
	}
}

func newHTTPExporter(ctx context.Context, cfg Config) (sdklog.Exporter, error) {
	var opts []otlploghttp.Option
	opts = append(opts, otlploghttp.WithEndpoint(cfg.Endpoint))
	if cfg.Insecure {
		opts = append(opts, otlploghttp.WithInsecure())
	}
	if len(cfg.Headers) > 0 {
		opts = append(opts, otlploghttp.WithHeaders(cfg.Headers))
	}
	return otlploghttp.New(ctx, opts...)
}

func newGRPCExporter(ctx context.Context, cfg Config) (sdklog.Exporter, error) {
	var opts []otlploggrpc.Option
	opts = append(opts, otlploggrpc.WithEndpoint(cfg.Endpoint))
	if cfg.Insecure {
		opts = append(opts, otlploggrpc.WithInsecure())
	}
	if len(cfg.Headers) > 0 {
		opts = append(opts, otlploggrpc.WithHeaders(cfg.Headers))
	}
	return otlploggrpc.New(ctx, opts...)
}
