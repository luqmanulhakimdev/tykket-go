package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/luqmanulhakimdev/tykket-go/pkg/token"
)

type AuthProvider string

const (
	AuthProviderLocal  AuthProvider = "LOCAL"
	AuthProviderGoogle AuthProvider = "GOOGLE"
	AuthProviderGithub AuthProvider = "GITHUB"
)

type User struct {
	ID           uuid.UUID    `json:"id"`
	Email        string       `json:"email"`
	Name         string       `json:"name"`
	PasswordHash *string      `json:"-"`
	AuthProvider AuthProvider `json:"auth_provider"`
	ProviderID   *string      `json:"provider_id,omitempty"`
	AvatarURL    *string      `json:"avatar_url,omitempty"`
	CreatedAt    time.Time    `json:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at"`
	DeletedAt    *time.Time   `json:"deleted_at,omitempty"`
}

type RegisterRequest struct {
	Name     string `json:"name" validate:"required,min=2,max=150"`
	Email    string `json:"email" validate:"required,email,max=255"`
	Password string `json:"password" validate:"required,min=8,max=72"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type GoogleAuthRequest struct {
	IDToken string `json:"id_token" validate:"required"`
}

type AuthResponse struct {
	User   *User            `json:"user"`
	Tokens *token.TokenPair `json:"tokens"`
}

type UserRepository interface {
	Create(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByProvider(ctx context.Context, provider AuthProvider, providerID string) (*User, error)
	Update(ctx context.Context, user *User) error
}

type AuthUsecase interface {
	Register(ctx context.Context, req RegisterRequest) (*AuthResponse, error)
	Login(ctx context.Context, req LoginRequest) (*AuthResponse, error)
	RefreshToken(ctx context.Context, req RefreshTokenRequest) (*AuthResponse, error)
	GoogleLogin(ctx context.Context, req GoogleAuthRequest) (*AuthResponse, error)
	GetMe(ctx context.Context, userID uuid.UUID) (*User, error)
}

