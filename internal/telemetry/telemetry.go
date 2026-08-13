package telemetry

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/company/service-registry/internal/config"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/semconv/v1.21.0"
)

// NewTracerProvider creates and returns a new OpenTelemetry TracerProvider
func NewTracerProvider(cfg config.TelemetryConfig) (*trace.TracerProvider, error) {
	if !cfg.Enabled {
		slog.Info("Telemetry disabled")
		return trace.NewTracerProvider(), nil
	}

	// Create resource
	res, err := resource.New(context.Background(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(cfg.ServiceName),
			semconv.ServiceVersionKey.String(cfg.ServiceVersion),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// TODO: Add OTLP exporter
	// For now, create a NoOp tracer provider
	tp := trace.NewTracerProvider(
		trace.WithResource(res),
	)

	// Set as global tracer provider
	otel.SetTracerProvider(tp)

	slog.Info("Telemetry initialized",
		slog.String("service", cfg.ServiceName),
		slog.String("version", cfg.ServiceVersion),
	)

	return tp, nil
}
