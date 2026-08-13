package auth

import (
	"context"
	"net/http"
	"strings"
)

type contextKey string

const principalKey contextKey = "auth.principal"

func WithPrincipal(ctx context.Context, principal *Principal) context.Context {
	return context.WithValue(ctx, principalKey, principal)
}

func PrincipalFromContext(ctx context.Context) (*Principal, bool) {
	principal, ok := ctx.Value(principalKey).(*Principal)
	return principal, ok
}

func Middleware(enabled bool, service *Service, next http.Handler) http.Handler {
	if !enabled {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isPublicPath(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}
		secret := bearerToken(r.Header.Get("Authorization"))
		if secret == "" {
			http.Error(w, "missing bearer token", http.StatusUnauthorized)
			return
		}
		principal, err := service.AuthenticateToken(r.Context(), secret)
		if err != nil {
			http.Error(w, "invalid bearer token", http.StatusUnauthorized)
			return
		}
		required := RequiredScope(r.Method, r.URL.Path)
		if !HasScope(principal.Scopes, required) {
			http.Error(w, "insufficient scope", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r.WithContext(WithPrincipal(r.Context(), principal)))
	})
}

func RequiredScope(method, path string) Scope {
	if strings.HasPrefix(path, "/api/v1/auth/users") || strings.HasPrefix(path, "/api/v1/auth/tokens") {
		return ScopeAdmin
	}
	if strings.Contains(path, "/List") || strings.Contains(path, "/Get") || strings.Contains(path, "/Watch") || method == http.MethodGet {
		return ScopeRead
	}
	if strings.Contains(path, "/CreateUser") || strings.Contains(path, "/CreateAPIToken") || strings.Contains(path, "/RevokeAPIToken") {
		return ScopeAdmin
	}
	return ScopeWrite
}

func isPublicPath(path string) bool {
	return path == "/api/v1/auth/login" || path == "/api/v1/ping"
}

func bearerToken(header string) string {
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(header, prefix))
}
