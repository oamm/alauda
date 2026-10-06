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
	return MiddlewareWithCookieName(enabled, service, "alauda_session", next)
}

func MiddlewareWithCookieName(enabled bool, service *Service, cookieName string, next http.Handler) http.Handler {
	if !enabled {
		return next
	}
	if cookieName == "" {
		cookieName = "alauda_session"
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isPublicPath(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}
		secret := bearerToken(r.Header.Get("Authorization"))
		credentialType := "bearer"
		if secret == "" {
			if cookie, err := r.Cookie(cookieName); err == nil {
				secret = cookie.Value
				credentialType = "session"
			}
		}
		if secret == "" {
			writeAuthError(w, http.StatusUnauthorized, "missing authentication")
			return
		}
		principal, err := service.AuthenticateToken(r.Context(), secret)
		if strings.HasPrefix(secret, "ak_") {
			principal, err = service.AuthenticateApplicationKey(r.Context(), secret)
		}
		if credentialType == "session" {
			principal, err = service.AuthenticateSession(r.Context(), secret)
		} else if err != nil {
			// CLI clients may use the compatibility token returned by login; it is a hashed session secret.
			principal, err = service.AuthenticateSession(r.Context(), secret)
		}
		if err != nil {
			writeAuthError(w, http.StatusUnauthorized, "invalid authentication")
			return
		}
		if principal.MustChangePassword && !passwordChangeAllowed(r.URL.Path) {
			writeAuthError(w, http.StatusForbidden, "password_change_required")
			return
		}
		required := RequiredScope(r.Method, r.URL.Path)
		if !HasScope(principal.Scopes, required) {
			writeAuthError(w, http.StatusForbidden, "insufficient scope")
			return
		}
		next.ServeHTTP(w, r.WithContext(WithPrincipal(r.Context(), principal)))
	})
}

func RequiredScope(method, path string) Scope {
	if strings.HasPrefix(path, "/api/v1/auth/users") || strings.HasPrefix(path, "/api/v1/auth/tokens") || strings.HasPrefix(path, "/api/v1/auth/application-keys") || strings.HasPrefix(path, "/api/v1/auth/sessions") {
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

func passwordChangeAllowed(path string) bool {
	return path == "/api/v1/auth/me" || path == "/api/v1/auth/logout" || path == "/api/v1/auth/password"
}

func writeAuthError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(`{"error":"` + message + `"}`))
}

func bearerToken(header string) string {
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(header, prefix))
}
