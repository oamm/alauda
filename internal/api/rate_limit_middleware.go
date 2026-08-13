package api

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/company/service-registry/internal/config"
)

type rateLimiter struct {
	mu      sync.Mutex
	clients map[string]*rateBucket
	rate    float64
	burst   float64
	now     func() time.Time
}

type rateBucket struct {
	tokens   float64
	updated  time.Time
	lastSeen time.Time
}

func newRateLimitMiddleware(cfg config.RateLimitConfig) func(http.Handler) http.Handler {
	if !cfg.Enabled {
		return func(next http.Handler) http.Handler { return next }
	}
	requestsPerMinute := cfg.RequestsPerMinute
	if requestsPerMinute <= 0 {
		requestsPerMinute = 600
	}
	burst := cfg.Burst
	if burst <= 0 {
		burst = requestsPerMinute / 10
		if burst <= 0 {
			burst = 1
		}
	}
	limiter := &rateLimiter{
		clients: map[string]*rateBucket{},
		rate:    float64(requestsPerMinute) / 60,
		burst:   float64(burst),
		now:     time.Now,
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !limiter.allow(clientAddress(r)) {
				w.Header().Set("Retry-After", "60")
				http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func (l *rateLimiter) allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	bucket := l.clients[key]
	if bucket == nil {
		l.clients[key] = &rateBucket{tokens: l.burst - 1, updated: now, lastSeen: now}
		l.cleanup(now)
		return true
	}
	elapsed := now.Sub(bucket.updated).Seconds()
	bucket.tokens += elapsed * l.rate
	if bucket.tokens > l.burst {
		bucket.tokens = l.burst
	}
	bucket.updated = now
	bucket.lastSeen = now
	if bucket.tokens < 1 {
		return false
	}
	bucket.tokens--
	l.cleanup(now)
	return true
}

func (l *rateLimiter) cleanup(now time.Time) {
	for key, bucket := range l.clients {
		if now.Sub(bucket.lastSeen) > 10*time.Minute {
			delete(l.clients, key)
		}
	}
}

func clientAddress(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		return strings.TrimSpace(strings.Split(forwarded, ",")[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}
