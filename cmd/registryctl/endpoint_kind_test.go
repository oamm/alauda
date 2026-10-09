package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPublicRegistrationKindPaths(t *testing.T) {
	old := cliConfig
	defer func() { cliConfig = old }()
	calls := 0
	var received publicCLIRegistration
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		json.NewDecoder(r.Body).Decode(&received)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{}`))
	}))
	defer server.Close()
	run := func(extra ...string) error {
		cliConfig = defaultCLIConfig()
		cmd := newRootCommand()
		cmd.SetArgs(append([]string{"services", "register", "--name", "Database", "--environment", "dev", "--address", "host", "--port", "5432", "--quiet", "--server", server.URL}, extra...))
		return cmd.Execute()
	}
	for _, kind := range []string{"tcp", "udp", "grpc", "postgres", "redis", "custom"} {
		if err := run("--kind", kind); err != nil {
			t.Fatal(err)
		}
		if received.Endpoints[0].Kind == nil || *received.Endpoints[0].Kind != kind {
			t.Fatalf("%s kind: %+v", kind, received)
		}
		if received.Endpoints[0].Path != nil && *received.Endpoints[0].Path != "" {
			t.Fatalf("%s path: %+v", kind, received)
		}
		before := calls
		if err := run("--kind", kind, "--path", "/bad"); err == nil || !strings.Contains(err.Error(), "HTTP and HTTPS") {
			t.Fatalf("invalid path: %v", err)
		}
		if calls != before {
			t.Fatal("invalid request reached server")
		}
	}
	if err := run("--kind", "http", "--path", "/health"); err != nil {
		t.Fatal(err)
	}
	if *received.Endpoints[0].Path != "/health" {
		t.Fatal("HTTP path lost")
	}
	if err := run("--endpoint", "default,tcp,5432"); err != nil {
		t.Fatal(err)
	}
	if err := run("--endpoint", "default,tcp,5432,/bad"); err == nil {
		t.Fatal("repeat endpoint bypassed validation")
	}
	if got, err := publicResolvedValue(map[string]any{"instance": map[string]any{"address": "192.168.0.109"}, "endpoint": map[string]any{"kind": "POSTGRES", "port": float64(5432)}}); err != nil || got != "192.168.0.109:5432" {
		t.Fatalf("postgres value = %q, %v", got, err)
	}
	if got, err := publicResolvedValue(map[string]any{"instance": map[string]any{"address": "lynx-authentication.lynx"}, "endpoint": map[string]any{"kind": "GRPC", "port": float64(81)}}); err != nil || got != "lynx-authentication.lynx:81" {
		t.Fatalf("grpc value = %q, %v", got, err)
	}
	if got, err := publicResolvedValue(map[string]any{"instance": map[string]any{"address": "api.internal"}, "endpoint": map[string]any{"kind": "HTTP", "port": float64(8080), "path": "/"}}); err != nil || got != "http://api.internal:8080/" {
		t.Fatalf("http value = %q, %v", got, err)
	}
}
