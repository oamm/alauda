package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestNormalizeList(t *testing.T) {
	got := normalizeList([]string{" unhealthy ", "", "recovered", "  "})
	want := []string{"unhealthy", "recovered"}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d: %v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestParseIncidentStateFlag(t *testing.T) {
	tests := []struct {
		input   string
		wantErr bool
	}{
		{"", false},
		{" open ", false},
		{"RESOLVED", false},
		{"closed", true},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			_, err := parseIncidentStateFlag(tt.input)
			if tt.wantErr && err == nil {
				t.Fatalf("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestDoJSONSuccessAndError(t *testing.T) {
	oldConfig := cliConfig
	defer func() { cliConfig = oldConfig }()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "application/json" {
			t.Fatalf("content-type = %q, want application/json", r.Header.Get("Content-Type"))
		}
		if r.URL.Path == "/fail" {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"ok": "true"})
	}))
	defer srv.Close()

	cliConfig = &config{ServerURL: srv.URL, Timeout: time.Second}
	var out map[string]string
	if err := doJSON(http.MethodPost, "/ok", strings.NewReader(`{}`), &out); err != nil {
		t.Fatalf("doJSON success: %v", err)
	}
	if out["ok"] != "true" {
		t.Fatalf("out = %v, want ok=true", out)
	}
	if err := doJSON(http.MethodPost, "/fail", strings.NewReader(`{}`), nil); err == nil {
		t.Fatalf("expected error response")
	}
}

func TestAuthTransportUsesConfiguredAndEnvironmentToken(t *testing.T) {
	oldConfig := cliConfig
	oldToken := os.Getenv("REGISTRY_TOKEN")
	defer func() {
		cliConfig = oldConfig
		_ = os.Setenv("REGISTRY_TOKEN", oldToken)
	}()

	tests := []struct {
		name        string
		configToken string
		envToken    string
		want        string
	}{
		{name: "configured token wins", configToken: "config-token", envToken: "env-token", want: "Bearer config-token"},
		{name: "environment token fallback", envToken: "env-token", want: "Bearer env-token"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cliConfig = &config{Token: tt.configToken, Timeout: time.Second}
			_ = os.Setenv("REGISTRY_TOKEN", tt.envToken)

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if got := r.Header.Get("Authorization"); got != tt.want {
					t.Fatalf("authorization = %q, want %q", got, tt.want)
				}
				w.WriteHeader(http.StatusNoContent)
			}))
			defer srv.Close()

			resp, err := newHTTPClient().Get(srv.URL)
			if err != nil {
				t.Fatalf("get: %v", err)
			}
			_ = resp.Body.Close()
		})
	}
}

func TestAlertCommandValidation(t *testing.T) {
	tests := []struct {
		name string
		cmd  interface{ Execute() error }
	}{
		{name: "alert policy missing scope", cmd: newAlertPolicyCreateCommand()},
		{name: "notification channel missing type and name", cmd: newNotificationChannelCreateCommand()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.cmd.Execute(); err == nil {
				t.Fatalf("expected validation error")
			}
		})
	}
}
