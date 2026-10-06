package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/company/service-registry/internal/auth"
	"github.com/company/service-registry/internal/storage"
)

type authHandler struct {
	service    *auth.Service
	repo       *auth.Repository
	cookieName string
	audit      auth.AuditLogger
}

func registerAuthREST(mux *http.ServeMux, handler authHandler) {
	mux.HandleFunc("/api/v1/auth/login", handler.login)
	mux.HandleFunc("/api/v1/auth/logout", handler.logout)
	mux.HandleFunc("/api/v1/auth/password", handler.password)
	mux.HandleFunc("/api/v1/auth/me", handler.me)
	mux.HandleFunc("/api/v1/auth/sessions", handler.sessions)
	mux.HandleFunc("/api/v1/auth/sessions/", handler.sessionByID)
	mux.HandleFunc("/api/v1/auth/users", handler.users)
	mux.HandleFunc("/api/v1/auth/tokens", handler.tokens)
	mux.HandleFunc("/api/v1/auth/tokens/", handler.tokenByID)
	mux.HandleFunc("/api/v1/auth/application-keys", handler.applicationKeys)
	mux.HandleFunc("/api/v1/auth/application-keys/", handler.applicationKeyByID)
}

func (h authHandler) login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var input struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid json body", http.StatusBadRequest)
		return
	}
	created, user, err := h.service.LoginSession(r.Context(), input.Username, input.Password, r.UserAgent(), clientAddress(r))
	if err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) {
			h.auditLogin(r, input.Username, false)
			http.Error(w, "invalid credentials", http.StatusUnauthorized)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.auditLogin(r, user.Username, true)
	secure := r.TLS != nil
	http.SetCookie(w, &http.Cookie{Name: h.cookie(), Value: created.Secret, Path: "/", HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode, Expires: created.Session.ExpiresAt})
	// Keep the token field for registryctl and other automation clients; browsers use the HttpOnly cookie.
	writeJSON(w, http.StatusOK, map[string]any{"user": user, "token": created.Secret, "expiresAt": created.Session.ExpiresAt, "mustChangePassword": user.MustChangePassword})
}

func (h authHandler) auditLogin(r *http.Request, username string, success bool) {
	if h.audit == nil {
		return
	}
	status, message := "failure", "invalid credentials"
	if success {
		status, message = "success", "authenticated"
	}
	entry := storage.AuditLog{
		Timestamp:         time.Now().UTC(),
		Actor:             username,
		Action:            "login",
		ResourceType:      "authentication",
		ResourceID:        "-",
		ChangeDescription: message,
		IP:                clientAddress(r),
		UserAgent:         r.UserAgent(),
		Status:            status,
		ErrorMessage:      message,
	}
	if success {
		entry.ErrorMessage = ""
	}
	_, _ = h.audit.Create(r.Context(), entry)
}

func (h authHandler) logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	principal, ok := auth.PrincipalFromContext(r.Context())
	if ok && principal.SessionID != "" {
		_ = h.repo.RevokeSession(r.Context(), principal.SessionID)
	}
	http.SetCookie(w, &http.Cookie{Name: h.cookie(), Value: "", Path: "/", HttpOnly: true, MaxAge: -1, SameSite: http.SameSiteLaxMode})
	w.WriteHeader(http.StatusNoContent)
}

func (h authHandler) password(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthenticated", http.StatusUnauthorized)
		return
	}
	var input struct {
		NewPassword     string `json:"newPassword"`
		ConfirmPassword string `json:"confirmPassword"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid json body", http.StatusBadRequest)
		return
	}
	if len(input.NewPassword) < 12 || input.NewPassword != input.ConfirmPassword {
		http.Error(w, "password must be at least 12 characters and match confirmation", http.StatusBadRequest)
		return
	}
	if err := h.repo.ChangePassword(r.Context(), principal.UserID, input.NewPassword); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if principal.SessionID != "" {
		_ = h.repo.RevokeSession(r.Context(), principal.SessionID)
	}
	http.SetCookie(w, &http.Cookie{Name: h.cookie(), Value: "", Path: "/", HttpOnly: true, MaxAge: -1, SameSite: http.SameSiteLaxMode})
	w.WriteHeader(http.StatusNoContent)
}

func (h authHandler) cookie() string {
	if h.cookieName != "" {
		return h.cookieName
	}
	return "alauda_session"
}

func (h authHandler) sessions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	sessions, err := h.repo.ListSessions(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"sessions": sessions})
}

func (h authHandler) sessionByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/auth/sessions/")
	if id == "" {
		http.Error(w, "session id is required", http.StatusBadRequest)
		return
	}
	if err := h.repo.RevokeSession(r.Context(), id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "session not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h authHandler) me(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthenticated", http.StatusUnauthorized)
		return
	}
	user, err := h.repo.GetUser(r.Context(), principal.UserID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"user": user, "scopes": principal.Scopes, "mustChangePassword": user.MustChangePassword})
}

func (h authHandler) users(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		users, err := h.repo.ListUsers(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"users": users})
	case http.MethodPost:
		var input struct {
			Username    string            `json:"username"`
			Email       string            `json:"email"`
			DisplayName string            `json:"displayName"`
			Password    string            `json:"password"`
			Role        auth.Role         `json:"role"`
			Tags        map[string]string `json:"tags"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, "invalid json body", http.StatusBadRequest)
			return
		}
		if input.Username == "" || input.Email == "" || input.DisplayName == "" || input.Password == "" {
			http.Error(w, "username, email, displayName, and password are required", http.StatusBadRequest)
			return
		}
		user, err := h.repo.CreateUser(r.Context(), auth.CreateUserInput{
			Username:    input.Username,
			Email:       input.Email,
			DisplayName: input.DisplayName,
			Password:    input.Password,
			Role:        input.Role,
			Tags:        input.Tags,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"user": user})
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (h authHandler) tokens(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		principal, ok := auth.PrincipalFromContext(r.Context())
		userID := r.URL.Query().Get("userId")
		if userID == "" && ok {
			userID = principal.UserID
		}
		if userID == "" {
			http.Error(w, "userId is required", http.StatusBadRequest)
			return
		}
		tokens, err := h.repo.ListTokens(r.Context(), userID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"tokens": tokens})
	case http.MethodPost:
		principal, _ := auth.PrincipalFromContext(r.Context())
		var input struct {
			UserID         string       `json:"userId"`
			Name           string       `json:"name"`
			Scopes         []auth.Scope `json:"scopes"`
			EnvironmentIDs []string     `json:"environmentIds"`
			ExpiresAt      *time.Time   `json:"expiresAt"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, "invalid json body", http.StatusBadRequest)
			return
		}
		if input.UserID == "" || input.Name == "" {
			http.Error(w, "userId and name are required", http.StatusBadRequest)
			return
		}
		created, err := h.service.CreateToken(r.Context(), principal, auth.CreateTokenInput{
			UserID:         input.UserID,
			Name:           input.Name,
			Scopes:         input.Scopes,
			EnvironmentIDs: input.EnvironmentIDs,
			ExpiresAt:      input.ExpiresAt,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		writeJSON(w, http.StatusCreated, created)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (h authHandler) tokenByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/auth/tokens/")
	if id == "" {
		http.Error(w, "token id is required", http.StatusBadRequest)
		return
	}
	if err := h.repo.RevokeToken(r.Context(), id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "token not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h authHandler) applicationKeys(w http.ResponseWriter, r *http.Request) {
	principal, _ := auth.PrincipalFromContext(r.Context())
	switch r.Method {
	case http.MethodGet:
		keys, err := h.repo.ListApplicationKeys(r.Context())
		if err != nil {
			http.Error(w, "failed to list application keys", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"keys": keys})
	case http.MethodPost:
		var input struct {
			Name           string       `json:"name"`
			Scopes         []auth.Scope `json:"scopes"`
			EnvironmentIDs []string     `json:"environmentIds"`
			ExpiresAt      *time.Time   `json:"expiresAt"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil || strings.TrimSpace(input.Name) == "" {
			http.Error(w, "name is required", http.StatusBadRequest)
			return
		}
		created, err := h.service.CreateApplicationKey(r.Context(), principal, auth.CreateApplicationKeyInput{
			Name: input.Name, Scopes: input.Scopes, EnvironmentIDs: input.EnvironmentIDs, ExpiresAt: input.ExpiresAt,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		writeJSON(w, http.StatusCreated, created)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (h authHandler) applicationKeyByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/auth/application-keys/")
	if id == "" {
		http.Error(w, "key id is required", http.StatusBadRequest)
		return
	}
	if err := h.repo.RevokeApplicationKey(r.Context(), id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "application key not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to revoke application key", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
