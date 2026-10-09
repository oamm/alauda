package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPublicRegistrationProtocolPaths(t *testing.T) {
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
	for _, protocol := range []string{"tcp", "udp", "grpc"} {
		if err := run("--protocol", protocol); err != nil {
			t.Fatal(err)
		}
		if received.Endpoints[0].Path != nil && *received.Endpoints[0].Path != "" {
			t.Fatalf("%s path: %+v", protocol, received)
		}
		before := calls
		if err := run("--protocol", protocol, "--path", "/bad"); err == nil || !strings.Contains(err.Error(), "HTTP and HTTPS") {
			t.Fatalf("invalid path: %v", err)
		}
		if calls != before {
			t.Fatal("invalid request reached server")
		}
	}
	if err := run("--protocol", "http", "--path", "/health"); err != nil {
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
}
