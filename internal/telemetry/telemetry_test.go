package telemetry

import (
	"testing"

	"github.com/company/service-registry/internal/config"
	"go.opentelemetry.io/otel"
)

func TestNewTracerProviderDisabled(t *testing.T) {
	tp, err := NewTracerProvider(config.TelemetryConfig{Enabled: false})
	if err != nil {
		t.Fatalf("NewTracerProvider() error = %v", err)
	}
	if tp == nil {
		t.Fatalf("expected tracer provider")
	}
}

func TestNewTracerProviderEnabledSetsGlobalProvider(t *testing.T) {
	tp, err := NewTracerProvider(config.TelemetryConfig{
		Enabled:        true,
		ServiceName:    "registry-test",
		ServiceVersion: "test",
	})
	if err != nil {
		t.Fatalf("NewTracerProvider() error = %v", err)
	}
	if tp == nil {
		t.Fatalf("expected tracer provider")
	}
	if got := otel.GetTracerProvider(); got != tp {
		t.Fatalf("global tracer provider was not set")
	}
}
