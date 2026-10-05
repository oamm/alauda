package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/company/service-registry/internal/config"
)

func TestRateLimitMiddlewareRejectsAfterBurst(t *testing.T) {
	middleware := newRateLimitMiddleware(config.RateLimitConfig{
		Enabled:           true,
		RequestsPerMinute: 60,
		Burst:             2,
	})
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/ping", nil)
		req.RemoteAddr = "10.0.0.1:1234"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("request %d status = %d, want %d", i, rec.Code, http.StatusNoContent)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/ping", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusTooManyRequests)
	}
}

func TestRateLimiterRefillsTokens(t *testing.T) {
	now := time.Now()
	limiter := &rateLimiter{
		clients: map[string]*rateBucket{},
		rate:    1,
		burst:   1,
		now:     func() time.Time { return now },
	}

	if !limiter.allow("client") {
		t.Fatalf("first request should be allowed")
	}
	if limiter.allow("client") {
		t.Fatalf("second request should be rejected before refill")
	}
	now = now.Add(time.Second)
	if !limiter.allow("client") {
		t.Fatalf("request should be allowed after refill")
	}
}

func TestClientAddressIgnoresSpoofedForwardedHeader(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/ping", nil)
	req.RemoteAddr = "10.0.0.7:1234"
	req.Header.Set("X-Forwarded-For", "192.0.2.99")

	if got := clientAddress(req); got != "10.0.0.7" {
		t.Fatalf("client address = %q, want socket peer address", got)
	}
}
