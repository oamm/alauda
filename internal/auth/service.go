package auth

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

var ErrInvalidCredentials = errors.New("invalid credentials")

type Service struct {
	repo     *Repository
	tokenTTL time.Duration
}

func NewService(repo *Repository, tokenTTL time.Duration) *Service {
	return &Service{repo: repo, tokenTTL: tokenTTL}
}

func (s *Service) Login(ctx context.Context, username, password string) (*CreatedToken, *User, error) {
	user, err := s.repo.GetUserByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil, ErrInvalidCredentials
		}
		return nil, nil, err
	}
	if !user.Enabled || !CheckPassword(user.PasswordHash, password) {
		return nil, nil, ErrInvalidCredentials
	}
	expiresAt := time.Now().UTC().Add(s.tokenTTL)
	created, err := s.repo.CreateToken(ctx, CreateTokenInput{
		UserID:    user.ID,
		Name:      "session",
		Scopes:    RoleScopes(user.Role),
		ExpiresAt: &expiresAt,
		CreatedBy: user.ID,
	})
	if err != nil {
		return nil, nil, err
	}
	_ = s.repo.TouchLogin(ctx, user.ID)
	return created, user, nil
}

func (s *Service) AuthenticateToken(ctx context.Context, secret string) (*Principal, error) {
	token, user, err := s.repo.FindTokenBySecret(ctx, secret)
	if err != nil {
		return nil, err
	}
	return &Principal{
		UserID:         user.ID,
		Username:       user.Username,
		Role:           user.Role,
		TokenID:        token.ID,
		Scopes:         token.Scopes,
		EnvironmentIDs: token.EnvironmentIDs,
	}, nil
}

func (s *Service) CreateToken(ctx context.Context, principal *Principal, input CreateTokenInput) (*CreatedToken, error) {
	if principal == nil || !HasScope(principal.Scopes, ScopeAdmin) {
		return nil, errors.New("admin scope is required")
	}
	if len(input.Scopes) == 0 {
		input.Scopes = []Scope{ScopeRead}
	}
	input.CreatedBy = principal.UserID
	return s.repo.CreateToken(ctx, input)
}
