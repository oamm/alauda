package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/company/service-registry/internal/api"
	"github.com/company/service-registry/internal/auth"
	appconfig "github.com/company/service-registry/internal/config"
	"github.com/company/service-registry/internal/storage"
	"gopkg.in/yaml.v3"
)

func captureCLI(t *testing.T, args ...string) (string, error) {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	defer func() { os.Stdout = old; r.Close() }()
	output := make(chan string, 1)
	go func() { data, _ := io.ReadAll(r); output <- string(data) }()
	cmd := newRootCommand()
	cmd.SetArgs(args)
	runErr := cmd.Execute()
	w.Close()
	return <-output, runErr
}

func TestAuthenticatedPublicCLIWorkflow(t *testing.T) {
	old := cliConfig
	defer func() { cliConfig = old }()
	ctx := context.Background()
	db, err := storage.NewDatabase(ctx, ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.GetDB().SetMaxOpenConns(1)
	if err = storage.RunMigrations(ctx, db); err != nil {
		t.Fatal(err)
	}
	key, err := auth.NewRepository(db).CreateApplicationKey(ctx, auth.CreateApplicationKeyInput{Name: "test", Scopes: []auth.Scope{auth.ScopeAdmin}, CreatedBy: "test"})
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	api.RegisterRoutesWithConfig(mux, db, &appconfig.Config{Auth: appconfig.AuthConfig{Enabled: true}})
	server := httptest.NewServer(mux)
	defer server.Close()
	t.Setenv("ALAUDA_TOKEN", key.Secret)
	t.Setenv("ALAUDA_ENVIRONMENT", "stg")
	run := func(args ...string) (string, error) {
		cliConfig = defaultCLIConfig()
		return captureCLI(t, append(args, "--server", server.URL, "--config", "")...)
	}
	require := func(args ...string) string {
		t.Helper()
		out, err := run(args...)
		if err != nil {
			t.Fatalf("%v: %v", args, err)
		}
		if strings.Contains(out, key.Secret) || strings.Contains(strings.ToLower(out), "deployment") {
			t.Fatal("public workflow leaks internal detail or credential")
		}
		return out
	}
	require("environments", "create", "--key", "stg", "--quiet")
	require("services", "create", "--name", "Authentication.Grpc", "--quiet")
	args := []string{"services", "register", "--name", "Authentication.Grpc", "--address", "lynx-authentication.lynx", "--port", "81", "--output", "json"}
	first := require(args...)
	second := require(args...)
	var a, b struct {
		Instance struct {
			ID string `json:"id"`
		}
		Endpoints []struct {
			ID string `json:"id"`
		}
	}
	if json.Unmarshal([]byte(first), &a) != nil || json.Unmarshal([]byte(second), &b) != nil || a.Instance.ID == "" || a.Instance.ID != b.Instance.ID || a.Endpoints[0].ID != b.Endpoints[0].ID {
		t.Fatal("rerun identity not stable")
	}
	value := require("services", "resolve", "Authentication.Grpc", "--output", "value", "--quiet")
	if value != "http://lynx-authentication.lynx:81\n" {
		t.Fatalf("value=%q", value)
	}
	for _, format := range []string{"json", "yaml", "table"} {
		out := require("services", "endpoints", "Authentication.Grpc", "--instance", "lynx-authentication.lynx", "--output", format)
		if strings.Contains(out, "%!") {
			t.Fatal("table artifact")
		}
		if format == "json" {
			var dto any
			if json.Unmarshal([]byte(out), &dto) != nil {
				t.Fatal("invalid json")
			}
		}
		if format == "yaml" {
			var dto any
			if yaml.Unmarshal([]byte(out), &dto) != nil {
				t.Fatal("invalid YAML")
			}
		}
	}
	serviceList := require("services", "list", "--environment", "stg", "--output", "json")
	if !strings.Contains(serviceList, `"healthStatus":"Unknown"`) {
		t.Fatalf("unmonitored Service must report canonical Unknown health: %s", serviceList)
	}
	serviceTable := require("services", "list", "--environment", "stg", "--output", "table")
	if !strings.Contains(serviceTable, "HEALTHSTATUS") || !strings.Contains(serviceTable, "Unknown") {
		t.Fatalf("table omitted canonical health status: %s", serviceTable)
	}
	require("services", "instances", "Authentication.Grpc", "--output", "table", "--page-size", "1")
	if _, err = run("services", "endpoints", "Authentication.Grpc", "--instance", "missing"); exitCode(err) != 5 {
		t.Fatalf("not found=%v", err)
	}
	if _, err = run("services", "register", "--name", "Authentication.Grpc", "--address", "host", "--port", "65536"); exitCode(err) != 2 {
		t.Fatalf("validation=%v", err)
	}
	if _, err = run("services", "resolve", "Authentication.Grpc", "--token", "invalid", "--output", "value"); exitCode(err) != 3 {
		t.Fatalf("auth=%v", err)
	}
	require("services", "deregister", "--name", "Authentication.Grpc", "--instance", "lynx-authentication.lynx", "--yes")
	require("services", "deregister", "--name", "Authentication.Grpc", "--instance", "lynx-authentication.lynx", "--yes")
	if _, err = run("services", "resolve", "Authentication.Grpc", "--output", "value"); exitCode(err) != 7 {
		t.Fatalf("candidate failure=%v", err)
	}
}
