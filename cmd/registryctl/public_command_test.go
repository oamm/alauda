package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRegistrationContractAndPrecedence(t *testing.T) {
	old := cliConfig
	defer func() { cliConfig = old }()
	t.Setenv("ALAUDA_ENVIRONMENT", "stg")
	var received publicCLIRegistration
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/services/Authentication.Grpc/instances" {
			t.Errorf("path=%s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Error(err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{}`))
	}))
	defer server.Close()
	run := func(args ...string) error {
		cliConfig = defaultCLIConfig()
		cmd := newRootCommand()
		cmd.SetArgs(append(args, "--server", server.URL, "--quiet"))
		return cmd.Execute()
	}
	if err := run("services", "register", "--name", "Authentication.Grpc", "--address", "auth-host", "--port", "81"); err != nil {
		t.Fatal(err)
	}
	if received.Instance.Name != "auth-host" || received.Environment != "stg" || received.Endpoints[0].Name != "default" || *received.Endpoints[0].Port != 81 || received.Endpoints[0].Primary != nil {
		t.Fatalf("%+v", received)
	}
	path := filepath.Join(t.TempDir(), "registration.yaml")
	// Test fixtures are written at runtime; production file registration uses the shared DTO.
	if err := os.WriteFile(path, []byte("service: Authentication.Grpc\ninstance:\n  name: auth-01\n  address: auth-host\n  enabled: false\n  tags: {}\nendpoints:\n  - name: metrics\n    protocol: http\n    port: 9090\n    primary: false\n    enabled: false\nreplaceEndpoints: true\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := run("services", "register", "--file", path, "--environment", "prod", "--address", "override-host"); err != nil {
		t.Fatal(err)
	}
	if received.Environment != "prod" || *received.Instance.Address != "override-host" || *received.Instance.Enabled || received.Instance.Tags == nil || *received.Endpoints[0].Primary || !received.Replace {
		t.Fatalf("lost file fields: %+v", received)
	}
	for _, port := range []string{"0", "65536", "4294967377", "9223372036854775808"} {
		if err := run("services", "register", "--name", "Authentication.Grpc", "--address", "auth-host", "--port", port); err == nil {
			t.Fatalf("accepted %s", port)
		}
	}
	for _, port := range []string{"1", "65535"} {
		if _, err := parsePublicPort(port); err != nil {
			t.Fatal(err)
		}
	}
	if err := run("services", "register", "--name", "Authentication.Grpc", "--address", "auth-host", "--endpoint", "default,http,81,/", "--endpoint", "metrics,http,9090,/metrics"); err != nil {
		t.Fatal(err)
	}
	if len(received.Endpoints) != 2 {
		t.Fatal("multiple endpoints lost")
	}
	if err := run("services", "list", "--environment", "stg", "--output", "invalid"); err == nil {
		t.Fatal("invalid output accepted")
	}
}

func TestExitCodesAndRedaction(t *testing.T) {
	for status, want := range map[int]int{400: 2, 401: 3, 403: 4, 404: 5, 409: 6, 500: 8} {
		if got := exitCode(publicCLIError{Status: status}); got != want {
			t.Fatalf("%d => %d", status, got)
		}
	}
	if exitCode(publicCLIError{Status: 404, Code: "no_healthy_instance"}) != 7 {
		t.Fatal("candidate exit code")
	}
	t.Setenv("ALAUDA_TOKEN", "sentinel-secret")
	if strings.Contains(safeError(publicCLIError{Detail: "sentinel-secret"}), "sentinel-secret") {
		t.Fatal("error leaked credential")
	}
	if publicCell(nil) != "" {
		t.Fatal("nil table cell")
	}
}

func TestNonsecretConfigAndFileEnvironmentPrecedence(t *testing.T) {
	old := cliConfig
	defer func() { cliConfig = old }()
	t.Setenv("ALAUDA_URL", "")
	t.Setenv("ALAUDA_ENVIRONMENT", "")
	var received publicCLIRegistration
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&received)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()
	dir := t.TempDir()
	cfg := filepath.Join(dir, "config.yaml")
	file := filepath.Join(dir, "service.yaml")
	if err := os.WriteFile(cfg, []byte("server: "+server.URL+"\nenvironment: configured\noutput: yaml\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte("service: Authentication.Grpc\ninstance:\n  name: auth\n  address: host\nendpoints:\n  - name: default\n    protocol: http\n    port: 81\n"), 0600); err != nil {
		t.Fatal(err)
	}
	run := func(flags ...string) {
		t.Helper()
		cliConfig = defaultCLIConfig()
		_, err := captureCLI(t, append([]string{"services", "register", "--file", file, "--config", cfg, "--quiet"}, flags...)...)
		if err != nil {
			t.Fatal(err)
		}
	}
	run()
	if received.Environment != "configured" || cliConfig.Output != "yaml" {
		t.Fatal("config fallback ignored")
	}
	t.Setenv("ALAUDA_ENVIRONMENT", "from-env")
	run("--output", "json")
	if received.Environment != "from-env" || cliConfig.Output != "json" {
		t.Fatal("environment/output override ignored")
	}
	if err := os.WriteFile(file, []byte("service: Authentication.Grpc\nenvironment: from-file\ninstance:\n  name: auth\n  address: host\nendpoints:\n  - name: default\n    protocol: http\n    port: 81\n"), 0600); err != nil {
		t.Fatal(err)
	}
	run()
	if received.Environment != "from-file" {
		t.Fatal("file precedence ignored")
	}
	run("--environment", "explicit")
	if received.Environment != "explicit" {
		t.Fatal("flag precedence ignored")
	}
	if err := os.WriteFile(cfg, []byte("token: sentinel-secret\n"), 0600); err != nil {
		t.Fatal(err)
	}
	cliConfig = defaultCLIConfig()
	_, err := captureCLI(t, "services", "list", "--config", cfg)
	if err == nil || strings.Contains(err.Error(), "sentinel-secret") {
		t.Fatal("secret config accepted or echoed")
	}
}

func TestMalformedServerResponseIsServerFailure(t *testing.T) {
	old := cliConfig
	defer func() { cliConfig = old }()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("malformed server response")) }))
	defer server.Close()
	cliConfig = defaultCLIConfig()
	cliConfig.ServerURL = server.URL
	var output any
	if err := doJSON("GET", "/api/v1/services", nil, &output); exitCode(err) != 8 {
		t.Fatalf("malformed response exit: %v", err)
	}
}
