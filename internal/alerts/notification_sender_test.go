package alerts

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/smtp"
	"strings"
	"testing"

	registryv1 "github.com/company/service-registry/gen/go/api/registry/v1"
)

func TestSenderSendsWebhookJSONPayload(t *testing.T) {
	var payload map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Fatalf("content-type = %q, want application/json", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	result := NewSender(server.Client()).Send(context.Background(), &registryv1.NotificationChannel{
		Id:      "channel-1",
		Type:    "webhook",
		Name:    "Ops",
		Enabled: true,
		Configuration: map[string]string{
			"url": server.URL,
		},
	}, Notification{
		Type: "unhealthy",
		Incident: &registryv1.Incident{
			Id:         "incident-1",
			InstanceId: "instance-1",
			Reason:     "health check failed",
		},
		Policy: &registryv1.AlertPolicy{Id: "policy-1"},
	})
	if result.ErrorMessage != "" {
		t.Fatalf("send result error = %q", result.ErrorMessage)
	}
	if result.StatusCode != http.StatusAccepted {
		t.Fatalf("status code = %d, want %d", result.StatusCode, http.StatusAccepted)
	}
	if payload["type"] != "unhealthy" || payload["incident_id"] != "incident-1" || payload["policy_id"] != "policy-1" {
		t.Fatalf("unexpected payload: %+v", payload)
	}
}

func TestSenderSendsEmailTemplate(t *testing.T) {
	sender := NewSender(nil)
	var gotAddr string
	var gotFrom string
	var gotTo []string
	var gotMessage string
	sender.sendMail = func(addr string, _ smtp.Auth, from string, to []string, msg []byte) error {
		gotAddr = addr
		gotFrom = from
		gotTo = to
		gotMessage = string(msg)
		return nil
	}

	result := sender.Send(context.Background(), &registryv1.NotificationChannel{
		Id:   "channel-1",
		Type: "email",
		Name: "Email",
		Configuration: map[string]string{
			"smtp_host": "smtp.example.test",
			"smtp_port": "2525",
			"from":      "registry@example.test",
			"to":        "ops@example.test,dev@example.test",
		},
	}, Notification{
		Type: "recovered",
		Incident: &registryv1.Incident{
			Id:         "incident-1",
			InstanceId: "instance-1",
			Reason:     "health check recovered",
		},
		Policy: &registryv1.AlertPolicy{Id: "policy-1"},
	})
	if result.ErrorMessage != "" {
		t.Fatalf("send result error = %q", result.ErrorMessage)
	}
	if gotAddr != "smtp.example.test:2525" {
		t.Fatalf("addr = %q, want smtp.example.test:2525", gotAddr)
	}
	if gotFrom != "registry@example.test" {
		t.Fatalf("from = %q, want registry@example.test", gotFrom)
	}
	if len(gotTo) != 2 || gotTo[0] != "ops@example.test" || gotTo[1] != "dev@example.test" {
		t.Fatalf("recipients = %v, want ops and dev", gotTo)
	}
	if !strings.Contains(gotMessage, "Subject: Service Registry alert: recovered") || !strings.Contains(gotMessage, "Incident incident-1 for instance instance-1") {
		t.Fatalf("unexpected message: %s", gotMessage)
	}
}
