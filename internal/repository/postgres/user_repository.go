package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/luqmanulhakimdev/tykket-go/internal/domain"
)

type userRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) domain.UserRepository {
	return &userRepository{pool: pool}
}

func (r *userRepository) Create(ctx context.Context, u *domain.User) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}

	query := `
		INSERT INTO users (id, email, name, password_hash, auth_provider, provider_id, avatar_url, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
		RETURNING created_at, updated_at
	`

	err := r.pool.QueryRow(ctx, query,
		u.ID,
		u.Email,
		u.Name,
		u.PasswordHash,
		u.AuthProvider,
		u.ProviderID,
		u.AvatarURL,
	).Scan(&u.CreatedAt, &u.UpdatedAt)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.ErrConflict
		}
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

func (r *userRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	query := `
		SELECT id, email, name, password_hash, auth_provider, provider_id, avatar_url, created_at, updated_at, deleted_at
		FROM users
		WHERE id = $1 AND deleted_at IS NULL
	`
	var u domain.User
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&u.ID,
		&u.Email,
		&u.Name,
		&u.PasswordHash,
		&u.AuthProvider,
		&u.ProviderID,
		&u.AvatarURL,
		&u.CreatedAt,
		&u.UpdatedAt,
		&u.DeletedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get user by id: %w", err)
	}
	return &u, nil
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `
		SELECT id, email, name, password_hash, auth_provider, provider_id, avatar_url, created_at, updated_at, deleted_at
		FROM users
		WHERE LOWER(email) = LOWER($1) AND deleted_at IS NULL
	`
	var u domain.User
	err := r.pool.QueryRow(ctx, query, email).Scan(
		&u.ID,
		&u.Email,
		&u.Name,
		&u.PasswordHash,
		&u.AuthProvider,
		&u.ProviderID,
		&u.AvatarURL,
		&u.CreatedAt,
		&u.UpdatedAt,
		&u.DeletedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}
	return &u, nil
}

func (r *userRepository) GetByProvider(ctx context.Context, provider domain.AuthProvider, providerID string) (*domain.User, error) {
	query := `
		SELECT id, email, name, password_hash, auth_provider, provider_id, avatar_url, created_at, updated_at, deleted_at
		FROM users
		WHERE auth_provider = $1 AND provider_id = $2 AND deleted_at IS NULL
	`
	var u domain.User
	err := r.pool.QueryRow(ctx, query, provider, providerID).Scan(
		&u.ID,
		&u.Email,
		&u.Name,
		&u.PasswordHash,
		&u.AuthProvider,
		&u.ProviderID,
		&u.AvatarURL,
		&u.CreatedAt,
		&u.UpdatedAt,
		&u.DeletedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get user by provider: %w", err)
	}
	return &u, nil
}

func (r *userRepository) Update(ctx context.Context, u *domain.User) error {
	query := `
		UPDATE users
		SET name = $1, avatar_url = $2, updated_at = NOW()
		WHERE id = $3 AND deleted_at IS NULL
		RETURNING updated_at
	`
	err := r.pool.QueryRow(ctx, query, u.Name, u.AvatarURL, u.ID).Scan(&u.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrNotFound
		}
		return fmt.Errorf("failed to update user: %w", err)
	}
	return nil
}

