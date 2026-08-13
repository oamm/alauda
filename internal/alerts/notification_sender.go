package alerts

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/smtp"
	"strings"
	"time"

	registryv1 "github.com/company/service-registry/gen/go/api/registry/v1"
)

type Notification struct {
	Type     string
	Incident *registryv1.Incident
	Policy   *registryv1.AlertPolicy
	Test     bool
}

type SendResult struct {
	StatusCode   int
	ErrorMessage string
}

type Sender struct {
	client   *http.Client
	sendMail func(addr string, a smtp.Auth, from string, to []string, msg []byte) error
}

func NewSender(client *http.Client) *Sender {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	return &Sender{client: client, sendMail: smtp.SendMail}
}

func (s *Sender) Send(ctx context.Context, channel *registryv1.NotificationChannel, notification Notification) SendResult {
	switch strings.ToLower(channel.GetType()) {
	case "webhook":
		return s.sendWebhook(ctx, channel, notification)
	case "email":
		return s.sendEmail(channel, notification)
	default:
		return SendResult{ErrorMessage: fmt.Sprintf("unsupported channel type %q", channel.GetType())}
	}
}

func (s *Sender) sendWebhook(ctx context.Context, channel *registryv1.NotificationChannel, notification Notification) SendResult {
	url := channel.GetConfiguration()["url"]
	if url == "" {
		return SendResult{ErrorMessage: "webhook url is required"}
	}
	payload := map[string]any{
		"type":           notification.Type,
		"test":           notification.Test,
		"channel_id":     channel.GetId(),
		"channel_name":   channel.GetName(),
		"policy_id":      notification.Policy.GetId(),
		"incident_id":    notification.Incident.GetId(),
		"environment_id": notification.Incident.GetEnvironmentId(),
		"service_id":     notification.Incident.GetServiceId(),
		"deployment_id":  notification.Incident.GetDeploymentId(),
		"instance_id":    notification.Incident.GetInstanceId(),
		"reason":         notification.Incident.GetReason(),
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return SendResult{ErrorMessage: err.Error()}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return SendResult{ErrorMessage: err.Error()}
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return SendResult{ErrorMessage: err.Error()}
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return SendResult{StatusCode: resp.StatusCode, ErrorMessage: resp.Status}
	}
	return SendResult{StatusCode: resp.StatusCode}
}

func (s *Sender) sendEmail(channel *registryv1.NotificationChannel, notification Notification) SendResult {
	cfg := channel.GetConfiguration()
	host := cfg["smtp_host"]
	port := cfg["smtp_port"]
	from := cfg["from"]
	to := cfg["to"]
	if host == "" || port == "" || from == "" || to == "" {
		return SendResult{ErrorMessage: "smtp_host, smtp_port, from, and to are required"}
	}

	subject := fmt.Sprintf("Service Registry alert: %s", notification.Type)
	body := fmt.Sprintf("Incident %s for instance %s: %s", notification.Incident.GetId(), notification.Incident.GetInstanceId(), notification.Incident.GetReason())
	message := []byte("To: " + to + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"Content-Type: text/plain; charset=UTF-8\r\n\r\n" +
		body + "\r\n")

	var auth smtp.Auth
	if username := cfg["username"]; username != "" {
		auth = smtp.PlainAuth("", username, cfg["password"], host)
	}
	err := s.sendMail(host+":"+port, auth, from, emailRecipients(to), message)
	if err != nil {
		return SendResult{ErrorMessage: err.Error()}
	}
	return SendResult{}
}

func emailRecipients(value string) []string {
	raw := strings.Split(value, ",")
	recipients := make([]string, 0, len(raw))
	for _, recipient := range raw {
		recipient = strings.TrimSpace(recipient)
		if recipient != "" {
			recipients = append(recipients, recipient)
		}
	}
	return recipients
}

func SendResultError(result SendResult) error {
	if result.ErrorMessage == "" {
		return nil
	}
	return errors.New(result.ErrorMessage)
}
