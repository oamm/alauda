package api

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

const maxRequestBodyBytes int64 = 1 << 20

func hardeningMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startedAt := time.Now()
		ctx, span := otel.Tracer("service-registry/api").Start(r.Context(), r.Method+" "+r.URL.Path)
		defer span.End()
		r = r.WithContext(ctx)
		rec := &statusRecorder{ResponseWriter: w, statusCode: http.StatusOK}
		defer func() {
			if recovered := recover(); recovered != nil {
				slog.Error("Request panic", slog.Any("panic", recovered), slog.String("path", r.URL.Path))
				http.Error(rec, "internal server error", http.StatusInternalServerError)
			}
			span.SetAttributes(
				attribute.String("http.method", r.Method),
				attribute.String("http.route", r.URL.Path),
				attribute.Int("http.status_code", rec.statusCode),
			)
			slog.Info("Request completed",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", rec.statusCode),
				slog.Duration("duration", time.Since(startedAt)),
			)
		}()
		if r.Body != nil {
			r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
		}
		if expectsBody(r) && !allowedContentType(r.Header.Get("Content-Type")) {
			http.Error(w, "unsupported content-type", http.StatusUnsupportedMediaType)
			return
		}
		next.ServeHTTP(rec, r)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (w *statusRecorder) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func expectsBody(r *http.Request) bool {
	switch r.Method {
	case http.MethodPost, http.MethodPut, http.MethodPatch:
		return true
	default:
		return false
	}
}

func allowedContentType(contentType string) bool {
	if contentType == "" {
		return true
	}
	normalized := strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
	return normalized == "application/json" ||
		normalized == "application/proto" ||
		strings.HasPrefix(normalized, "application/connect+")
}
