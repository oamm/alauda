package api

import (
	"bytes"
	"net/http"
	"strings"

	"github.com/company/service-registry/internal/problem"
)

// Convert older REST and middleware failures without forwarding repository
// text. Success bodies and SSE writes remain streaming, never buffered.
func publicErrors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/v1/") {
			next.ServeHTTP(w, r)
			return
		}
		rec := &problemRecorder{ResponseWriter: w, status: 200}
		next.ServeHTTP(rec, r)
		if rec.status < 400 {
			return
		}
		if strings.HasPrefix(w.Header().Get("Content-Type"), "application/problem+json") {
			w.WriteHeader(rec.status)
			_, _ = w.Write(rec.body.Bytes())
			return
		}
		code := map[int]string{400: "validation_failed", 401: "authentication_required", 403: "permission_denied", 404: "resource_not_found", 405: "method_not_allowed", 409: "resource_conflict", 413: "request_too_large", 415: "unsupported_media_type", 429: "rate_limited", 503: "temporarily_unavailable"}[rec.status]
		if code == "" {
			code = "server_error"
		}
		problem.Write(w, rec.status, code, http.StatusText(rec.status), nil)
	})
}

type problemRecorder struct {
	http.ResponseWriter
	status int
	body   bytes.Buffer
}

func (w *problemRecorder) WriteHeader(status int) {
	w.status = status
	if status < 400 {
		w.ResponseWriter.WriteHeader(status)
	}
}
func (w *problemRecorder) Write(data []byte) (int, error) {
	if w.status >= 400 {
		return w.body.Write(data)
	}
	return w.ResponseWriter.Write(data)
}
func (w *problemRecorder) Flush() {
	if w.status < 400 {
		if f, ok := w.ResponseWriter.(http.Flusher); ok {
			f.Flush()
		}
	}
}
