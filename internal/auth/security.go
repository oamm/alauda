package auth

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

type BootstrapResult struct {
	Created        bool
	Username       string
	CredentialPath string
}

// ResetDevelopmentBootstrap rotates the root development credential. It is
// intentionally separate from Bootstrap so production initialization remains
// one-time and idempotent.
func (r *Repository) ResetDevelopmentBootstrap(ctx context.Context, username, credentialPath string) (*BootstrapResult, error) {
	if err := os.Remove(credentialPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("remove development bootstrap credential: %w", err)
	}
	password, err := newBootstrapCredential(credentialPath, username)
	if err != nil {
		return nil, err
	}
	hash, err := HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("hash development bootstrap credential: %w", err)
	}

	tx, err := r.db.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin development bootstrap reset: %w", err)
	}
	defer tx.Rollback()

	var userID string
	if err := tx.QueryRowContext(ctx, "SELECT id FROM users WHERE username = ? AND deleted_at IS NULL", username).Scan(&userID); err != nil {
		return nil, fmt.Errorf("find development bootstrap administrator: %w", err)
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := tx.ExecContext(ctx, "UPDATE users SET password_hash = ?, must_change_password = 1, updated_at = ? WHERE id = ?", hash, now, userID); err != nil {
		return nil, fmt.Errorf("reset development bootstrap password: %w", err)
	}
	if _, err := tx.ExecContext(ctx, "UPDATE sessions SET revoked_at = ? WHERE user_id = ? AND revoked_at IS NULL", now, userID); err != nil {
		return nil, fmt.Errorf("revoke development bootstrap sessions: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit development bootstrap reset: %w", err)
	}
	return &BootstrapResult{Created: true, Username: username, CredentialPath: credentialPath}, nil
}

// Bootstrap performs the one-time initialization under a database uniqueness boundary.
// Existing installations are marked initialized and are never given an implicit root.
func (r *Repository) Bootstrap(ctx context.Context, username, email, credentialPath string) (*BootstrapResult, error) {
	tx, err := r.db.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin security bootstrap: %w", err)
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(ctx, "INSERT OR IGNORE INTO security_bootstrap (id) VALUES (1)")
	if err != nil {
		return nil, fmt.Errorf("claim security bootstrap: %w", err)
	}
	claimed, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("inspect security bootstrap claim: %w", err)
	}
	if claimed == 0 {
		return &BootstrapResult{Username: username, CredentialPath: credentialPath}, tx.Commit()
	}

	var userCount int
	if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM users WHERE deleted_at IS NULL").Scan(&userCount); err != nil {
		return nil, fmt.Errorf("inspect existing users: %w", err)
	}
	if userCount > 0 {
		if _, err := tx.ExecContext(ctx, "UPDATE security_bootstrap SET initialized_at = ?, root_user_id = NULL WHERE id = 1", time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
			return nil, fmt.Errorf("mark existing installation initialized: %w", err)
		}
		return &BootstrapResult{Username: username, CredentialPath: credentialPath}, tx.Commit()
	}

	password, err := bootstrapCredential(credentialPath, username)
	if err != nil {
		return nil, err
	}
	hash, err := HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("hash bootstrap credential: %w", err)
	}
	rootID := uuid.NewString()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO users (id, username, email, display_name, password_hash, role, enabled, must_change_password, tags, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, 1, 1, '{}', ?, ?)
	`, rootID, username, email, "Root Administrator", hash, string(RoleAdministrator), now, now); err != nil {
		return nil, fmt.Errorf("create bootstrap administrator: %w", err)
	}
	if _, err := tx.ExecContext(ctx, "UPDATE security_bootstrap SET initialized_at = ?, root_user_id = ? WHERE id = 1", now, rootID); err != nil {
		return nil, fmt.Errorf("complete security bootstrap: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit security bootstrap: %w", err)
	}
	return &BootstrapResult{Created: true, Username: username, CredentialPath: credentialPath}, nil
}

func bootstrapCredential(path, username string) (string, error) {
	if data, err := os.ReadFile(path); err == nil {
		parts := strings.SplitN(strings.TrimSpace(string(data)), "\n", 2)
		if len(parts) == 2 && strings.TrimSpace(parts[0]) == "username: "+username && strings.TrimSpace(parts[1]) != "" {
			return strings.TrimSpace(parts[1]), nil
		}
		return "", fmt.Errorf("bootstrap credential file %s has an invalid format", path)
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("read bootstrap credential file: %w", err)
	}

	return newBootstrapCredential(path, username)
}

func newBootstrapCredential(path, username string) (string, error) {
	secret, _, err := NewToken()
	if err != nil {
		return "", fmt.Errorf("generate bootstrap credential: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return "", fmt.Errorf("create bootstrap credential directory: %w", err)
	}
	contents := "username: " + username + "\n" + secret + "\n"
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err == nil {
		_ = file.Chmod(0o600)
		if _, writeErr := file.WriteString(contents); writeErr != nil {
			file.Close()
			return "", fmt.Errorf("write bootstrap credential: %w", writeErr)
		}
		if closeErr := file.Close(); closeErr != nil {
			return "", fmt.Errorf("close bootstrap credential: %w", closeErr)
		}
		return secret, nil
	}
	if !errors.Is(err, os.ErrExist) {
		return "", fmt.Errorf("create bootstrap credential file: %w", err)
	}
	data, readErr := os.ReadFile(path)
	if readErr != nil {
		return "", fmt.Errorf("read concurrent bootstrap credential: %w", readErr)
	}
	parts := strings.SplitN(strings.TrimSpace(string(data)), "\n", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[1]) == "" {
		return "", fmt.Errorf("concurrent bootstrap credential file has an invalid format")
	}
	return strings.TrimSpace(parts[1]), nil
}

func (r *Repository) CreateSession(ctx context.Context, userID string, ttl time.Duration, userAgent, clientIP string) (*CreatedSession, error) {
	secret, hash, err := NewToken()
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	session := &Session{ID: uuid.NewString(), UserID: userID, CreatedAt: now, LastActivityAt: now, ExpiresAt: now.Add(ttl), UserAgent: userAgent, ClientIP: clientIP}
	_, err = r.db.Exec(ctx, `INSERT INTO sessions (id, user_id, secret_hash, created_at, last_activity_at, expires_at, user_agent, client_ip) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, session.ID, userID, hash, now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano), session.ExpiresAt.Format(time.RFC3339Nano), userAgent, clientIP)
	if err != nil {
		return nil, err
	}
	return &CreatedSession{Session: session, Secret: secret}, nil
}

func (r *Repository) FindSessionBySecret(ctx context.Context, secret string) (*Session, *User, error) {
	row := r.db.QueryRow(ctx, `SELECT s.id, s.user_id, s.created_at, s.last_activity_at, s.expires_at, s.revoked_at, s.user_agent, s.client_ip,
		u.id, u.username, u.email, u.display_name, u.password_hash, u.role, u.enabled, u.must_change_password, u.tags, u.created_at, u.updated_at, u.last_login_at
		FROM sessions s JOIN users u ON u.id = s.user_id AND u.deleted_at IS NULL WHERE s.secret_hash = ?`, HashToken(secret))
	var session Session
	var user User
	var created, activity, expires, revoked, tags, userCreated, userUpdated, lastLogin, userAgent, clientIP sql.NullString
	if err := row.Scan(&session.ID, &session.UserID, &created, &activity, &expires, &revoked, &userAgent, &clientIP, &user.ID, &user.Username, &user.Email, &user.DisplayName, &user.PasswordHash, &user.Role, &user.Enabled, &user.MustChangePassword, &tags, &userCreated, &userUpdated, &lastLogin); err != nil {
		return nil, nil, err
	}
	session.CreatedAt, session.LastActivityAt, session.ExpiresAt = parseTime(created.String), parseTime(activity.String), parseTime(expires.String)
	session.UserAgent, session.ClientIP = userAgent.String, clientIP.String
	if revoked.Valid {
		value := parseTime(revoked.String)
		session.RevokedAt = &value
	}
	user.Tags = map[string]string{}
	_ = jsonUnmarshal(tags.String, &user.Tags)
	user.CreatedAt, user.UpdatedAt = parseTime(userCreated.String), parseTime(userUpdated.String)
	if lastLogin.Valid {
		value := parseTime(lastLogin.String)
		user.LastLoginAt = &value
	}
	if !user.Enabled || session.RevokedAt != nil || time.Now().UTC().After(session.ExpiresAt) {
		return nil, nil, sql.ErrNoRows
	}
	_, _ = r.db.Exec(ctx, "UPDATE sessions SET last_activity_at = ? WHERE id = ?", time.Now().UTC().Format(time.RFC3339Nano), session.ID)
	return &session, &user, nil
}

func (r *Repository) RevokeSession(ctx context.Context, id string) error {
	result, err := r.db.Exec(ctx, "UPDATE sessions SET revoked_at = ? WHERE id = ? AND revoked_at IS NULL", time.Now().UTC().Format(time.RFC3339Nano), id)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err == nil && count == 0 {
		return sql.ErrNoRows
	}
	return err
}

func (r *Repository) ListSessions(ctx context.Context) ([]*Session, error) {
	rows, err := r.db.Query(ctx, `SELECT id, user_id, created_at, last_activity_at, expires_at, revoked_at, user_agent, client_ip FROM sessions ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var sessions []*Session
	for rows.Next() {
		var item Session
		var created, activity, expires, revoked, userAgent, clientIP sql.NullString
		if err := rows.Scan(&item.ID, &item.UserID, &created, &activity, &expires, &revoked, &userAgent, &clientIP); err != nil {
			return nil, err
		}
		item.CreatedAt, item.LastActivityAt, item.ExpiresAt = parseTime(created.String), parseTime(activity.String), parseTime(expires.String)
		item.UserAgent, item.ClientIP = userAgent.String, clientIP.String
		if revoked.Valid {
			value := parseTime(revoked.String)
			item.RevokedAt = &value
		}
		sessions = append(sessions, &item)
	}
	return sessions, rows.Err()
}

func (r *Repository) ChangePassword(ctx context.Context, userID, password string) error {
	hash, err := HashPassword(password)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(ctx, "UPDATE users SET password_hash = ?, must_change_password = 0, updated_at = ? WHERE id = ?", hash, time.Now().UTC().Format(time.RFC3339Nano), userID)
	return err
}

func jsonUnmarshal(raw string, target any) error {
	return json.Unmarshal([]byte(raw), target)
}
