package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/company/service-registry/internal/storage"
)

type AuditLogger interface {
	Create(ctx context.Context, entry storage.AuditLog) (*storage.AuditLog, error)
}

func AuditMiddleware(logger AuditLogger, next http.Handler) http.Handler {
	if logger == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		required := RequiredScope(r.Method, r.URL.Path)
		if required == ScopeRead || isPublicPath(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		var body []byte
		if r.Body != nil {
			body, _ = io.ReadAll(io.LimitReader(r.Body, 64*1024))
			r.Body.Close()
			r.Body = io.NopCloser(bytes.NewReader(body))
		}
		rec := &auditResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		startedAt := time.Now().UTC()
		next.ServeHTTP(rec, r)

		principal, _ := PrincipalFromContext(r.Context())
		entry := storage.AuditLog{
			Timestamp:         startedAt,
			Actor:             actorName(principal),
			ActorID:           actorID(principal),
			Action:            auditAction(r),
			ResourceType:      auditResourceType(r.URL.Path),
			ResourceID:        auditResourceID(r.URL.Path),
			Changes:           auditChanges(body),
			ChangeDescription: r.Method + " " + r.URL.Path,
			IP:                clientIP(r),
			UserAgent:         r.UserAgent(),
			Status:            auditStatus(rec.statusCode),
			Metadata: map[string]string{
				"method":     r.Method,
				"path":       r.URL.Path,
				"statusCode": statusCodeText(rec.statusCode),
			},
		}
		if rec.statusCode >= 400 {
			entry.ErrorMessage = http.StatusText(rec.statusCode)
		}
		if _, err := logger.Create(context.Background(), entry); err != nil {
			slog.Warn("Failed to write audit log", slog.String("error", err.Error()))
		}
	})
}

type auditResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *auditResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func auditStatus(statusCode int) string {
	if statusCode >= 200 && statusCode < 400 {
		return "success"
	}
	return "failure"
}

func auditAction(r *http.Request) string {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) > 0 {
		last := parts[len(parts)-1]
		if strings.Contains(last, "Service") {
			methodParts := strings.Split(last, "/")
			return methodParts[len(methodParts)-1]
		}
	}
	switch r.Method {
	case http.MethodPost:
		return "create"
	case http.MethodPut, http.MethodPatch:
		return "update"
	case http.MethodDelete:
		return "delete"
	default:
		return strings.ToLower(r.Method)
	}
}

func auditResourceType(path string) string {
	trimmed := strings.Trim(path, "/")
	if strings.HasPrefix(trimmed, "api/v1/auth/") {
		return "auth"
	}
	if strings.HasPrefix(trimmed, "api/v1/alerts/") {
		return "alert"
	}
	parts := strings.Split(trimmed, ".")
	if len(parts) >= 3 {
		servicePart := parts[len(parts)-1]
		if slash := strings.Index(servicePart, "/"); slash > 0 {
			return strings.TrimSuffix(servicePart[:slash], "Service")
		}
	}
	return "http"
}

func auditResourceID(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) >= 4 && parts[0] == "api" {
		return parts[len(parts)-1]
	}
	return "-"
}

func auditChanges(body []byte) string {
	if len(body) == 0 {
		return ""
	}
	var payload any
	if err := json.Unmarshal(body, &payload); err != nil {
		return string(body)
	}
	redactValue(payload)
	sanitized, err := json.Marshal(payload)
	if err != nil {
		return ""
	}
	return string(sanitized)
}

func redactValue(value any) {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			if sensitiveAuditKey(key) {
				typed[key] = "[REDACTED]"
				continue
			}
			redactValue(child)
		}
	case []any:
		for _, child := range typed {
			redactValue(child)
		}
	}
}

func sensitiveAuditKey(key string) bool {
	normalized := strings.ToLower(key)
	return strings.Contains(normalized, "password") ||
		strings.Contains(normalized, "token") ||
		strings.Contains(normalized, "secret") ||
		strings.Contains(normalized, "authorization")
}

func statusCodeText(statusCode int) string {
	text := http.StatusText(statusCode)
	if text == "" {
		return "unknown"
	}
	return text
}

func actorName(principal *Principal) string {
	if principal == nil || principal.Username == "" {
		return "anonymous"
	}
	return principal.Username
}

func actorID(principal *Principal) string {
	if principal == nil {
		return ""
	}
	return principal.UserID
}

func clientIP(r *http.Request) string {
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		return strings.TrimSpace(strings.Split(forwarded, ",")[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}
