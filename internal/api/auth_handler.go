package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/company/service-registry/internal/auth"
)

type authHandler struct {
	service *auth.Service
	repo    *auth.Repository
}

func registerAuthREST(mux *http.ServeMux, handler authHandler) {
	mux.HandleFunc("/api/v1/auth/login", handler.login)
	mux.HandleFunc("/api/v1/auth/me", handler.me)
	mux.HandleFunc("/api/v1/auth/users", handler.users)
	mux.HandleFunc("/api/v1/auth/tokens", handler.tokens)
	mux.HandleFunc("/api/v1/auth/tokens/", handler.tokenByID)
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
	created, user, err := h.service.Login(r.Context(), input.Username, input.Password)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) {
			http.Error(w, "invalid credentials", http.StatusUnauthorized)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"user": user, "token": created.Secret, "expiresAt": created.Token.ExpiresAt})
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
	writeJSON(w, http.StatusOK, map[string]any{"user": user, "scopes": principal.Scopes})
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

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
