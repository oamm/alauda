package health

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	registryv1 "github.com/company/service-registry/gen/go/api/registry/v1"
)

func TestExecutorHTTPCheckSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/readyz" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	target := targetFromServer(t, server, &registryv1.HealthCheck{
		Id:             "hc-1",
		InstanceId:     "inst-1",
		Type:           registryv1.HealthCheckType_HEALTH_CHECK_TYPE_HTTP,
		TimeoutSeconds: 2,
		Metadata:       map[string]string{"expectedStatus": "204"},
	})
	target.Path = "/readyz"

	result := NewExecutor(server.Client()).Execute(context.Background(), target)
	if !result.GetSuccess() {
		t.Fatalf("expected success, got error_type=%q error_message=%q", result.GetErrorType(), result.GetErrorMessage())
	}
	if result.GetStatusCode() != http.StatusNoContent {
		t.Fatalf("got status %d, want %d", result.GetStatusCode(), http.StatusNoContent)
	}
	if result.GetHealthCheckId() != "hc-1" || result.GetInstanceId() != "inst-1" {
		t.Fatalf("result ids were not copied from check: %#v", result)
	}
}

func TestExecutorHTTPCheckUnexpectedStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	result := NewExecutor(server.Client()).Execute(context.Background(), targetFromServer(t, server, &registryv1.HealthCheck{
		Type:           registryv1.HealthCheckType_HEALTH_CHECK_TYPE_HTTP,
		TimeoutSeconds: 2,
	}))

	if result.GetSuccess() {
		t.Fatalf("expected failure")
	}
	if result.GetErrorType() != "unexpected_status" {
		t.Fatalf("got error type %q, want unexpected_status", result.GetErrorType())
	}
}

func TestExecutorHTTPCheckTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	result := NewExecutor(server.Client()).Execute(context.Background(), targetFromServer(t, server, &registryv1.HealthCheck{
		Type:           registryv1.HealthCheckType_HEALTH_CHECK_TYPE_HTTP,
		TimeoutSeconds: 0,
	}))

	if !result.GetSuccess() {
		t.Fatalf("expected no per-check timeout when timeout_seconds is 0, got %q", result.GetErrorType())
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	result = NewExecutor(server.Client()).Execute(ctx, targetFromServer(t, server, &registryv1.HealthCheck{
		Type: registryv1.HealthCheckType_HEALTH_CHECK_TYPE_HTTP,
	}))

	if result.GetSuccess() {
		t.Fatalf("expected timeout failure")
	}
	if result.GetErrorType() != "timeout" {
		t.Fatalf("got error type %q, want timeout", result.GetErrorType())
	}
}

func TestExecutorTCPCheckSuccess(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer listener.Close()

	done := make(chan struct{})
	go func() {
		defer close(done)
		conn, err := listener.Accept()
		if err == nil {
			conn.Close()
		}
	}()

	host, port := splitListenerAddress(t, listener)
	result := NewExecutor(nil).Execute(context.Background(), Target{
		Check: &registryv1.HealthCheck{
			Id:             "hc-tcp",
			InstanceId:     "inst-tcp",
			Type:           registryv1.HealthCheckType_HEALTH_CHECK_TYPE_TCP,
			TimeoutSeconds: 2,
		},
		Address: host,
		Port:    port,
	})

	if !result.GetSuccess() {
		t.Fatalf("expected TCP success, got %q %q", result.GetErrorType(), result.GetErrorMessage())
	}
	<-done
}

func TestExecutorTCPCheckConnectionFailure(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	host, port := splitListenerAddress(t, listener)
	listener.Close()

	result := NewExecutor(nil).Execute(context.Background(), Target{
		Check: &registryv1.HealthCheck{
			Type:           registryv1.HealthCheckType_HEALTH_CHECK_TYPE_TCP,
			TimeoutSeconds: 1,
		},
		Address: host,
		Port:    port,
	})

	if result.GetSuccess() {
		t.Fatalf("expected TCP failure")
	}
	if result.GetErrorType() != "connection_error" {
		t.Fatalf("got error type %q, want connection_error", result.GetErrorType())
	}
}

func TestExecutorGRPCCheckUsesTransportReachability(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer listener.Close()

	done := make(chan struct{})
	go func() {
		defer close(done)
		conn, err := listener.Accept()
		if err == nil {
			conn.Close()
		}
	}()

	host, port := splitListenerAddress(t, listener)
	result := NewExecutor(nil).Execute(context.Background(), Target{
		Check: &registryv1.HealthCheck{
			Id:             "hc-grpc",
			InstanceId:     "inst-grpc",
			Type:           registryv1.HealthCheckType_HEALTH_CHECK_TYPE_GRPC,
			TimeoutSeconds: 2,
		},
		Address: host,
		Port:    port,
	})

	if !result.GetSuccess() {
		t.Fatalf("expected gRPC transport success, got %q %q", result.GetErrorType(), result.GetErrorMessage())
	}
	<-done
}

func TestExecutorRejectsInvalidTarget(t *testing.T) {
	result := NewExecutor(nil).Execute(context.Background(), Target{
		Check: &registryv1.HealthCheck{Type: registryv1.HealthCheckType_HEALTH_CHECK_TYPE_HTTP},
	})

	if result.GetSuccess() {
		t.Fatalf("expected failure")
	}
	if result.GetErrorType() != "invalid_target" {
		t.Fatalf("got error type %q, want invalid_target", result.GetErrorType())
	}
}

func splitListenerAddress(t *testing.T, listener net.Listener) (string, int32) {
	t.Helper()

	host, portRaw, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatalf("parse listener addr %q: %v", listener.Addr().String(), err)
	}
	port, err := strconv.Atoi(portRaw)
	if err != nil {
		t.Fatalf("parse listener port %q: %v", portRaw, err)
	}
	return host, int32(port)
}

func targetFromServer(t *testing.T, server *httptest.Server, check *registryv1.HealthCheck) Target {
	t.Helper()

	address := strings.TrimPrefix(server.URL, "http://")
	host, portRaw, err := net.SplitHostPort(address)
	if err != nil {
		t.Fatalf("parse server url %q: %v", server.URL, err)
	}
	port, err := strconv.Atoi(portRaw)
	if err != nil {
		t.Fatalf("parse server port %q: %v", portRaw, err)
	}

	return Target{
		Check:   check,
		Address: host,
		Port:    int32(port),
		Path:    "/",
	}
}
