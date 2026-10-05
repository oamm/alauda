package auth

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/company/service-registry/internal/storage"
	"github.com/google/uuid"
)

type Repository struct {
	db *storage.Database
}

func NewRepository(db *storage.Database) *Repository {
	return &Repository{db: db}
}

type CreateUserInput struct {
	Username           string
	Email              string
	DisplayName        string
	Password           string
	Role               Role
	Tags               map[string]string
	MustChangePassword bool
}

type CreateTokenInput struct {
	UserID         string
	Name           string
	Scopes         []Scope
	EnvironmentIDs []string
	ExpiresAt      *time.Time
	CreatedBy      string
	Token          string
}

type CreatedToken struct {
	Token  *APIToken `json:"token"`
	Secret string    `json:"secret"`
}

func (r *Repository) CreateUser(ctx context.Context, input CreateUserInput) (*User, error) {
	if !ValidRole(input.Role) {
		return nil, errors.New("invalid role")
	}
	passwordHash, err := HashPassword(input.Password)
	if err != nil {
		return nil, err
	}
	tagsJSON, err := json.Marshal(emptyStringMap(input.Tags))
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	id := uuid.NewString()
	_, err = r.db.Exec(ctx, `
		INSERT INTO users (id, username, email, display_name, password_hash, role, enabled, must_change_password, tags, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, 1, ?, ?, ?, ?)
	`, id, input.Username, input.Email, input.DisplayName, passwordHash, string(input.Role), input.MustChangePassword, string(tagsJSON), now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano))
	if err != nil {
		return nil, err
	}
	return r.GetUser(ctx, id)
}

func (r *Repository) GetUser(ctx context.Context, id string) (*User, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, username, email, display_name, password_hash, role, enabled, must_change_password, tags, created_at, updated_at, last_login_at
		FROM users
		WHERE id = ? AND deleted_at IS NULL
	`, id)
	return scanUser(row)
}

func (r *Repository) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, username, email, display_name, password_hash, role, enabled, must_change_password, tags, created_at, updated_at, last_login_at
		FROM users
		WHERE username = ? AND deleted_at IS NULL
	`, username)
	return scanUser(row)
}

func (r *Repository) ListUsers(ctx context.Context) ([]*User, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, username, email, display_name, password_hash, role, enabled, must_change_password, tags, created_at, updated_at, last_login_at
		FROM users
		WHERE deleted_at IS NULL
		ORDER BY username ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := []*User{}
	for rows.Next() {
		user, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, rows.Err()
}

func (r *Repository) TouchLogin(ctx context.Context, userID string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE users SET last_login_at = ?, updated_at = ? WHERE id = ?
	`, time.Now().UTC().Format(time.RFC3339Nano), time.Now().UTC().Format(time.RFC3339Nano), userID)
	return err
}

func (r *Repository) CreateToken(ctx context.Context, input CreateTokenInput) (*CreatedToken, error) {
	secret := input.Token
	tokenHash := ""
	var err error
	if secret == "" {
		secret, tokenHash, err = NewToken()
		if err != nil {
			return nil, err
		}
	} else {
		tokenHash = HashToken(secret)
	}
	scopesJSON, err := json.Marshal(input.Scopes)
	if err != nil {
		return nil, err
	}
	envJSON, err := json.Marshal(input.EnvironmentIDs)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	id := uuid.NewString()
	_, err = r.db.Exec(ctx, `
		INSERT INTO api_tokens (id, user_id, name, token_hash, scopes, environment_ids, expires_at, enabled, created_at, created_by)
		VALUES (?, ?, ?, ?, ?, ?, ?, 1, ?, ?)
	`, id, input.UserID, input.Name, tokenHash, string(scopesJSON), string(envJSON), nullTime(input.ExpiresAt), now.Format(time.RFC3339Nano), input.CreatedBy)
	if err != nil {
		return nil, err
	}
	token, err := r.GetToken(ctx, id)
	if err != nil {
		return nil, err
	}
	return &CreatedToken{Token: token, Secret: secret}, nil
}

func (r *Repository) GetToken(ctx context.Context, id string) (*APIToken, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, user_id, name, token_hash, scopes, environment_ids, expires_at, last_used_at, enabled, created_at, created_by
		FROM api_tokens
		WHERE id = ?
	`, id)
	return scanToken(row)
}

func (r *Repository) FindTokenBySecret(ctx context.Context, secret string) (*APIToken, *User, error) {
	row := r.db.QueryRow(ctx, `
		SELECT t.id, t.user_id, t.name, t.token_hash, t.scopes, t.environment_ids, t.expires_at, t.last_used_at,
		       t.enabled, t.created_at, t.created_by,
		       u.id, u.username, u.email, u.display_name, u.password_hash, u.role, u.enabled, u.must_change_password, u.tags,
		       u.created_at, u.updated_at, u.last_login_at
		FROM api_tokens t
		JOIN users u ON u.id = t.user_id AND u.deleted_at IS NULL
		WHERE t.token_hash = ?
	`, HashToken(secret))
	token, user, err := scanTokenAndUser(row)
	if err != nil {
		return nil, nil, err
	}
	if token.ExpiresAt != nil && time.Now().UTC().After(*token.ExpiresAt) {
		return nil, nil, sql.ErrNoRows
	}
	if !token.Enabled || !user.Enabled {
		return nil, nil, sql.ErrNoRows
	}
	_, _ = r.db.Exec(ctx, `UPDATE api_tokens SET last_used_at = ? WHERE id = ?`, time.Now().UTC().Format(time.RFC3339Nano), token.ID)
	return token, user, nil
}

func (r *Repository) ListTokens(ctx context.Context, userID string) ([]*APIToken, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, name, token_hash, scopes, environment_ids, expires_at, last_used_at, enabled, created_at, created_by
		FROM api_tokens
		WHERE user_id = ?
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	tokens := []*APIToken{}
	for rows.Next() {
		token, err := scanToken(rows)
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, token)
	}
	return tokens, rows.Err()
}

func (r *Repository) RevokeToken(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, `UPDATE api_tokens SET enabled = 0 WHERE id = ?`, id)
	return err
}

func (r *Repository) BootstrapAdmin(ctx context.Context, input CreateUserInput, tokenName, token string) (*CreatedToken, error) {
	user, err := r.GetUserByUsername(ctx, input.Username)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
		user, err = r.CreateUser(ctx, input)
		if err != nil {
			return nil, err
		}
	}
	if token == "" {
		return nil, nil
	}
	if _, _, err := r.FindTokenBySecret(ctx, token); err == nil {
		return nil, nil
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	return r.CreateToken(ctx, CreateTokenInput{
		UserID:    user.ID,
		Name:      tokenName,
		Scopes:    []Scope{ScopeRead, ScopeWrite, ScopeAdmin},
		CreatedBy: user.ID,
		Token:     token,
	})
}

func scanUser(row interface{ Scan(...any) error }) (*User, error) {
	var user User
	var tagsRaw string
	var createdRaw, updatedRaw string
	var lastLogin sql.NullString
	if err := row.Scan(&user.ID, &user.Username, &user.Email, &user.DisplayName, &user.PasswordHash, &user.Role, &user.Enabled, &user.MustChangePassword, &tagsRaw, &createdRaw, &updatedRaw, &lastLogin); err != nil {
		return nil, err
	}
	user.Tags = map[string]string{}
	_ = json.Unmarshal([]byte(tagsRaw), &user.Tags)
	user.CreatedAt = parseTime(createdRaw)
	user.UpdatedAt = parseTime(updatedRaw)
	if lastLogin.Valid {
		t := parseTime(lastLogin.String)
		user.LastLoginAt = &t
	}
	return &user, nil
}

func scanToken(row interface{ Scan(...any) error }) (*APIToken, error) {
	var token APIToken
	var scopesRaw string
	var envRaw sql.NullString
	var expiresRaw sql.NullString
	var lastUsedRaw sql.NullString
	var createdRaw string
	if err := row.Scan(&token.ID, &token.UserID, &token.Name, &token.TokenHash, &scopesRaw, &envRaw, &expiresRaw, &lastUsedRaw, &token.Enabled, &createdRaw, &token.CreatedBy); err != nil {
		return nil, err
	}
	_ = json.Unmarshal([]byte(scopesRaw), &token.Scopes)
	if envRaw.Valid {
		_ = json.Unmarshal([]byte(envRaw.String), &token.EnvironmentIDs)
	}
	if expiresRaw.Valid {
		t := parseTime(expiresRaw.String)
		token.ExpiresAt = &t
	}
	if lastUsedRaw.Valid {
		t := parseTime(lastUsedRaw.String)
		token.LastUsedAt = &t
	}
	token.CreatedAt = parseTime(createdRaw)
	return &token, nil
}

func scanTokenAndUser(row interface{ Scan(...any) error }) (*APIToken, *User, error) {
	var token APIToken
	var user User
	var scopesRaw string
	var envRaw, expiresRaw, lastUsedRaw sql.NullString
	var tokenCreatedRaw string
	var tagsRaw string
	var userCreatedRaw, userUpdatedRaw string
	var lastLoginRaw sql.NullString
	err := row.Scan(
		&token.ID, &token.UserID, &token.Name, &token.TokenHash, &scopesRaw, &envRaw, &expiresRaw, &lastUsedRaw,
		&token.Enabled, &tokenCreatedRaw, &token.CreatedBy,
		&user.ID, &user.Username, &user.Email, &user.DisplayName, &user.PasswordHash, &user.Role, &user.Enabled, &user.MustChangePassword, &tagsRaw,
		&userCreatedRaw, &userUpdatedRaw, &lastLoginRaw,
	)
	if err != nil {
		return nil, nil, err
	}
	_ = json.Unmarshal([]byte(scopesRaw), &token.Scopes)
	if envRaw.Valid {
		_ = json.Unmarshal([]byte(envRaw.String), &token.EnvironmentIDs)
	}
	if expiresRaw.Valid {
		t := parseTime(expiresRaw.String)
		token.ExpiresAt = &t
	}
	if lastUsedRaw.Valid {
		t := parseTime(lastUsedRaw.String)
		token.LastUsedAt = &t
	}
	token.CreatedAt = parseTime(tokenCreatedRaw)
	user.Tags = map[string]string{}
	_ = json.Unmarshal([]byte(tagsRaw), &user.Tags)
	user.CreatedAt = parseTime(userCreatedRaw)
	user.UpdatedAt = parseTime(userUpdatedRaw)
	if lastLoginRaw.Valid {
		t := parseTime(lastLoginRaw.String)
		user.LastLoginAt = &t
	}
	return &token, &user, nil
}

func emptyStringMap(value map[string]string) map[string]string {
	if value == nil {
		return map[string]string{}
	}
	return value
}

func nullTime(value *time.Time) any {
	if value == nil {
		return nil
	}
	return value.UTC().Format(time.RFC3339Nano)
}

func parseTime(raw string) time.Time {
	if t, err := time.Parse(time.RFC3339Nano, raw); err == nil {
		return t
	}
	return time.Time{}
}
