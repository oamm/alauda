package alerts

import (
	"context"
	"log/slog"
	"strconv"
	"time"

	registryv1 "github.com/company/service-registry/gen/go/api/registry/v1"
	"github.com/company/service-registry/internal/storage"
)

type Engine struct {
	repo   *storage.AlertRepository
	sender *Sender
	sleep  func(context.Context, time.Duration) error
}

func NewEngine(repo *storage.AlertRepository, sender *Sender) *Engine {
	if sender == nil {
		sender = NewSender(nil)
	}
	return &Engine{repo: repo, sender: sender, sleep: sleepContext}
}

func (e *Engine) ProcessIncident(ctx context.Context, incident *registryv1.Incident, notificationType string, at time.Time) error {
	policies, err := e.repo.MatchPolicies(ctx, incident, notificationType, at)
	if err != nil {
		return err
	}
	for _, policy := range policies {
		channels, err := e.repo.ListPolicyChannels(ctx, policy)
		if err != nil {
			return err
		}
		for _, channel := range channels {
			if err := e.sendWithRetry(ctx, channel, policy, incident, notificationType, at); err != nil {
				return err
			}
		}
	}
	return nil
}

func (e *Engine) sendWithRetry(ctx context.Context, channel *registryv1.NotificationChannel, policy *registryv1.AlertPolicy, incident *registryv1.Incident, notificationType string, at time.Time) error {
	maxAttempts := retryPolicyInt(channel.GetRetryPolicy(), "max_attempts", 1)
	backoff := time.Duration(retryPolicyInt(channel.GetRetryPolicy(), "backoff_ms", 500)) * time.Millisecond
	if maxAttempts < 1 {
		maxAttempts = 1
	}

	for attempt := 0; attempt < maxAttempts; attempt++ {
		result := e.sender.Send(ctx, channel, Notification{
			Type:     notificationType,
			Incident: incident,
			Policy:   policy,
		})
		success := result.ErrorMessage == ""
		var nextRetryAt *time.Time
		if !success && attempt+1 < maxAttempts {
			next := at.Add(backoffForAttempt(backoff, attempt))
			nextRetryAt = &next
		}
		if err := e.repo.LogAttempt(ctx, storage.AlertAttempt{
			IncidentID:       incident.GetId(),
			PolicyID:         policy.GetId(),
			ChannelID:        channel.GetId(),
			NotificationType: notificationType,
			AttemptedAt:      at,
			Success:          success,
			StatusCode:       result.StatusCode,
			ErrorMessage:     result.ErrorMessage,
			RetryCount:       attempt,
			NextRetryAt:      nextRetryAt,
		}); err != nil {
			return err
		}
		if success {
			return nil
		}
		slog.Warn("alert notification failed", slog.String("channel_id", channel.GetId()), slog.Int("attempt", attempt+1), slog.String("error", result.ErrorMessage))
		if attempt+1 < maxAttempts {
			if err := e.sleep(ctx, backoffForAttempt(backoff, attempt)); err != nil {
				return err
			}
		}
	}
	return nil
}

func (e *Engine) TestChannel(ctx context.Context, channel *registryv1.NotificationChannel) error {
	incident := &registryv1.Incident{
		Id:         "test",
		InstanceId: "test-instance",
		Reason:     "test notification",
	}
	policy := &registryv1.AlertPolicy{Id: "test"}
	result := e.sender.Send(ctx, channel, Notification{
		Type:     "test",
		Incident: incident,
		Policy:   policy,
		Test:     true,
	})
	return SendResultError(result)
}

func retryPolicyInt(policy map[string]string, key string, fallback int) int {
	value, ok := policy[key]
	if !ok || value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func backoffForAttempt(base time.Duration, attempt int) time.Duration {
	if base <= 0 {
		base = 500 * time.Millisecond
	}
	return base * time.Duration(1<<attempt)
}

func sleepContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
