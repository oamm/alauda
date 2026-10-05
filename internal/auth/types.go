package auth

import "time"

type User struct {
	ID                 string            `json:"id"`
	Username           string            `json:"username"`
	Email              string            `json:"email"`
	DisplayName        string            `json:"displayName"`
	PasswordHash       string            `json:"-"`
	Role               Role              `json:"role"`
	Enabled            bool              `json:"enabled"`
	MustChangePassword bool              `json:"mustChangePassword"`
	Tags               map[string]string `json:"tags,omitempty"`
	CreatedAt          time.Time         `json:"createdAt"`
	UpdatedAt          time.Time         `json:"updatedAt"`
	LastLoginAt        *time.Time        `json:"lastLoginAt,omitempty"`
}

type APIToken struct {
	ID             string     `json:"id"`
	UserID         string     `json:"userId"`
	Name           string     `json:"name"`
	TokenHash      string     `json:"-"`
	Scopes         []Scope    `json:"scopes"`
	EnvironmentIDs []string   `json:"environmentIds,omitempty"`
	ExpiresAt      *time.Time `json:"expiresAt,omitempty"`
	LastUsedAt     *time.Time `json:"lastUsedAt,omitempty"`
	Enabled        bool       `json:"enabled"`
	CreatedAt      time.Time  `json:"createdAt"`
	CreatedBy      string     `json:"createdBy"`
}

type Principal struct {
	UserID             string
	Username           string
	Role               Role
	TokenID            string
	Scopes             []Scope
	EnvironmentIDs     []string
	SessionID          string
	MustChangePassword bool
}

type Session struct {
	ID             string     `json:"id"`
	UserID         string     `json:"userId"`
	CreatedAt      time.Time  `json:"createdAt"`
	LastActivityAt time.Time  `json:"lastActivityAt"`
	ExpiresAt      time.Time  `json:"expiresAt"`
	RevokedAt      *time.Time `json:"revokedAt,omitempty"`
	UserAgent      string     `json:"userAgent,omitempty"`
	ClientIP       string     `json:"clientIp,omitempty"`
}

type CreatedSession struct {
	Session *Session
	Secret  string
}
