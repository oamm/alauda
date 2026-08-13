package main

import (
	"strings"
	"testing"

	registryv1 "github.com/company/service-registry/gen/go/api/registry/v1"
)

func TestParseHealthCheckTypeFlag(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    registryv1.HealthCheckType
		wantErr bool
	}{
		{name: "http", input: "http", want: registryv1.HealthCheckType_HEALTH_CHECK_TYPE_HTTP},
		{name: "case insensitive", input: "HTTPS", want: registryv1.HealthCheckType_HEALTH_CHECK_TYPE_HTTPS},
		{name: "trimmed", input: " tcp ", want: registryv1.HealthCheckType_HEALTH_CHECK_TYPE_TCP},
		{name: "empty", input: "", want: registryv1.HealthCheckType_HEALTH_CHECK_TYPE_UNSPECIFIED},
		{name: "invalid", input: "smtp", want: registryv1.HealthCheckType_HEALTH_CHECK_TYPE_UNSPECIFIED, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseHealthCheckTypeFlag(tt.input)
			if tt.wantErr && err == nil {
				t.Fatalf("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHealthCheckCreateValidation(t *testing.T) {
	cmd := newHealthCheckCreateCommand()
	cmd.SetArgs([]string{
		"--instance-id", "inst-1",
		"--name", "ready",
		"--type", "http",
		"--interval-seconds", "0",
	})

	err := cmd.Execute()
	if err == nil {
		t.Fatalf("expected validation error")
	}
	if !strings.Contains(err.Error(), "--interval-seconds must be greater than 0") {
		t.Fatalf("unexpected error: %v", err)
	}
}
