package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/luqmanulhakimdev/tykket-go/internal/domain"
	"github.com/luqmanulhakimdev/tykket-go/pkg/token"
	"golang.org/x/crypto/bcrypt"
)

type mockUserRepoForAuth struct {
	domain.UserRepository
	usersByEmail map[string]*domain.User
	usersByID    map[uuid.UUID]*domain.User
}

func (m *mockUserRepoForAuth) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	if u, ok := m.usersByEmail[email]; ok {
		return u, nil
	}
	return nil, domain.ErrNotFound
}

func (m *mockUserRepoForAuth) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	if u, ok := m.usersByID[id]; ok {
		return u, nil
	}
	return nil, domain.ErrNotFound
}

func (m *mockUserRepoForAuth) Create(ctx context.Context, user *domain.User) error {
	m.usersByEmail[user.Email] = user
	m.usersByID[user.ID] = user
	return nil
}

func (m *mockUserRepoForAuth) GetByProvider(ctx context.Context, provider domain.AuthProvider, providerID string) (*domain.User, error) {
	for _, u := range m.usersByID {
		if u.AuthProvider == provider && u.ProviderID != nil && *u.ProviderID == providerID {
			return u, nil
		}
	}
	return nil, domain.ErrNotFound
}

type mockTokenMgrForAuth struct {
	token.TokenManager
}

func (m *mockTokenMgrForAuth) GenerateTokenPair(userID uuid.UUID, email string) (*token.TokenPair, error) {
	return &token.TokenPair{
		AccessToken:  "mock-access-token",
		RefreshToken: "mock-refresh-token",
		ExpiresAt:    time.Now().Add(15 * time.Minute),
	}, nil
}

func (m *mockTokenMgrForAuth) ValidateToken(tokenString string, expectedType token.TokenType) (*token.Claims, error) {
	if tokenString == "valid-refresh-token" {
		return &token.Claims{
			UserID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			Email:  "test@example.com",
			Type:   token.RefreshToken,
		}, nil
	}
	return nil, errors.New("invalid token")
}

func TestAuthUsecase(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	hashStr := string(hash)

	existingUser := &domain.User{
		ID:           uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		Email:        "test@example.com",
		Name:         "Test User",
		PasswordHash: &hashStr,
		AuthProvider: domain.AuthProviderLocal,
	}

	oauthUser := &domain.User{
		ID:           uuid.MustParse("00000000-0000-0000-0000-000000000002"),
		Email:        "oauth@example.com",
		Name:         "OAuth User",
		PasswordHash: nil,
		AuthProvider: domain.AuthProviderGoogle,
	}

	userRepo := &mockUserRepoForAuth{
		usersByEmail: map[string]*domain.User{
			"test@example.com":  existingUser,
			"oauth@example.com": oauthUser,
		},
		usersByID: map[uuid.UUID]*domain.User{
			existingUser.ID: existingUser,
			oauthUser.ID:    oauthUser,
		},
	}
	tokenMgr := &mockTokenMgrForAuth{}
	uc := NewAuthUsecase(userRepo, tokenMgr)
	ctx := context.Background()

	t.Run("Register success", func(t *testing.T) {
		res, err := uc.Register(ctx, domain.RegisterRequest{
			Name:     "New User",
			Email:    "new@example.com",
			Password: "securepassword",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.User.Email != "new@example.com" {
			t.Errorf("expected email new@example.com, got %s", res.User.Email)
		}
	})

	t.Run("Register conflict email", func(t *testing.T) {
		_, err := uc.Register(ctx, domain.RegisterRequest{
			Name:     "Duplicate",
			Email:    "test@example.com",
			Password: "securepassword",
		})
		if !errors.Is(err, domain.ErrConflict) {
			t.Fatalf("expected ErrConflict, got %v", err)
		}
	})

	t.Run("Login success", func(t *testing.T) {
		res, err := uc.Login(ctx, domain.LoginRequest{
			Email:    "test@example.com",
			Password: "password123",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.Tokens.AccessToken != "mock-access-token" {
			t.Errorf("expected mock token, got %s", res.Tokens.AccessToken)
		}
	})

	t.Run("Login wrong password", func(t *testing.T) {
		_, err := uc.Login(ctx, domain.LoginRequest{
			Email:    "test@example.com",
			Password: "wrongpassword",
		})
		if !errors.Is(err, domain.ErrUnauthorized) {
			t.Fatalf("expected ErrUnauthorized, got %v", err)
		}
	})

	t.Run("Login user not found", func(t *testing.T) {
		_, err := uc.Login(ctx, domain.LoginRequest{
			Email:    "nonexistent@example.com",
			Password: "password123",
		})
		if !errors.Is(err, domain.ErrUnauthorized) {
			t.Fatalf("expected ErrUnauthorized, got %v", err)
		}
	})

	t.Run("Login OAuth user with password error", func(t *testing.T) {
		_, err := uc.Login(ctx, domain.LoginRequest{
			Email:    "oauth@example.com",
			Password: "password123",
		})
		if err == nil {
			t.Fatal("expected error logging into OAuth account with password, got nil")
		}
	})

	t.Run("RefreshToken success", func(t *testing.T) {
		res, err := uc.RefreshToken(ctx, domain.RefreshTokenRequest{
			RefreshToken: "valid-refresh-token",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.User.ID != existingUser.ID {
			t.Errorf("expected user ID %s, got %s", existingUser.ID, res.User.ID)
		}
	})

	t.Run("RefreshToken invalid token", func(t *testing.T) {
		_, err := uc.RefreshToken(ctx, domain.RefreshTokenRequest{
			RefreshToken: "garbage-token",
		})
		if !errors.Is(err, domain.ErrUnauthorized) {
			t.Fatalf("expected ErrUnauthorized, got %v", err)
		}
	})

	t.Run("GoogleLogin success", func(t *testing.T) {
		res, err := uc.GoogleLogin(ctx, domain.GoogleAuthRequest{
			IDToken: "valid-id-token",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.Tokens == nil {
			t.Error("expected tokens returned")
		}
	})

	t.Run("GoogleLogin empty token", func(t *testing.T) {
		_, err := uc.GoogleLogin(ctx, domain.GoogleAuthRequest{
			IDToken: "",
		})
		if !errors.Is(err, domain.ErrInvalidInput) {
			t.Fatalf("expected ErrInvalidInput, got %v", err)
		}
	})

	t.Run("GetMe success and not found", func(t *testing.T) {
		u, err := uc.GetMe(ctx, existingUser.ID)
		if err != nil || u.Email != existingUser.Email {
			t.Fatalf("failed to get user: %v", err)
		}

		_, err = uc.GetMe(ctx, uuid.New())
		if !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("expected ErrNotFound, got %v", err)
		}
	})
}
