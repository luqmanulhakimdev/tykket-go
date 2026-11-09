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

type projectRepository struct {
	pool *pgxpool.Pool
}

func NewProjectRepository(pool *pgxpool.Pool) domain.ProjectRepository {
	return &projectRepository{pool: pool}
}

func (r *projectRepository) Create(ctx context.Context, p *domain.Project) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}

	query := `
		INSERT INTO projects (id, name, key, description, owner_id, issue_counter, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, 0, NOW(), NOW())
		RETURNING created_at, updated_at
	`
	err := r.pool.QueryRow(ctx, query,
		p.ID,
		p.Name,
		p.Key,
		p.Description,
		p.OwnerID,
	).Scan(&p.CreatedAt, &p.UpdatedAt)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.ErrConflict
		}
		return fmt.Errorf("failed to create project: %w", err)
	}

	return nil
}

func (r *projectRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Project, error) {
	query := `
		SELECT 
			p.id, p.name, p.key, p.description, p.owner_id, p.issue_counter, p.created_at, p.updated_at, p.deleted_at,
			u.id, u.email, u.name, u.avatar_url
		FROM projects p
		JOIN users u ON p.owner_id = u.id AND u.deleted_at IS NULL
		WHERE p.id = $1 AND p.deleted_at IS NULL
	`
	var (
		p domain.Project
		u domain.User
	)
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&p.ID,
		&p.Name,
		&p.Key,
		&p.Description,
		&p.OwnerID,
		&p.IssueCounter,
		&p.CreatedAt,
		&p.UpdatedAt,
		&p.DeletedAt,
		&u.ID,
		&u.Email,
		&u.Name,
		&u.AvatarURL,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get project by id: %w", err)
	}
	p.Owner = &u
	return &p, nil
}

func (r *projectRepository) GetByKey(ctx context.Context, key string) (*domain.Project, error) {
	query := `
		SELECT 
			p.id, p.name, p.key, p.description, p.owner_id, p.issue_counter, p.created_at, p.updated_at, p.deleted_at,
			u.id, u.email, u.name, u.avatar_url
		FROM projects p
		JOIN users u ON p.owner_id = u.id AND u.deleted_at IS NULL
		WHERE UPPER(p.key) = UPPER($1) AND p.deleted_at IS NULL
	`
	var (
		p domain.Project
		u domain.User
	)
	err := r.pool.QueryRow(ctx, query, key).Scan(
		&p.ID,
		&p.Name,
		&p.Key,
		&p.Description,
		&p.OwnerID,
		&p.IssueCounter,
		&p.CreatedAt,
		&p.UpdatedAt,
		&p.DeletedAt,
		&u.ID,
		&u.Email,
		&u.Name,
		&u.AvatarURL,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get project by key: %w", err)
	}
	p.Owner = &u
	return &p, nil
}

func (r *projectRepository) ListByUserID(ctx context.Context, userID uuid.UUID) ([]domain.Project, error) {
	query := `
		SELECT DISTINCT
			p.id, p.name, p.key, p.description, p.owner_id, p.issue_counter, p.created_at, p.updated_at, p.deleted_at,
			u.id, u.email, u.name, u.avatar_url
		FROM projects p
		JOIN users u ON p.owner_id = u.id AND u.deleted_at IS NULL
		LEFT JOIN project_members pm ON p.id = pm.project_id AND pm.deleted_at IS NULL
		WHERE (p.owner_id = $1 OR pm.user_id = $1) AND p.deleted_at IS NULL
		ORDER BY p.updated_at DESC
	`
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}
	defer rows.Close()

	var projects []domain.Project
	for rows.Next() {
		var (
			p domain.Project
			u domain.User
		)
		if err := rows.Scan(
			&p.ID,
			&p.Name,
			&p.Key,
			&p.Description,
			&p.OwnerID,
			&p.IssueCounter,
			&p.CreatedAt,
			&p.UpdatedAt,
			&p.DeletedAt,
			&u.ID,
			&u.Email,
			&u.Name,
			&u.AvatarURL,
		); err != nil {
			return nil, fmt.Errorf("failed to scan project: %w", err)
		}
		p.Owner = &u
		projects = append(projects, p)
	}
	return projects, nil
}

func (r *projectRepository) Update(ctx context.Context, p *domain.Project) error {
	query := `
		UPDATE projects
		SET name = $1, description = $2, updated_at = NOW()
		WHERE id = $3 AND deleted_at IS NULL
		RETURNING updated_at
	`
	err := r.pool.QueryRow(ctx, query, p.Name, p.Description, p.ID).Scan(&p.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrNotFound
		}
		return fmt.Errorf("failed to update project: %w", err)
	}
	return nil
}

func (r *projectRepository) Delete(ctx context.Context, id uuid.UUID) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		UPDATE projects
		SET deleted_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`
	ct, err := tx.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to soft delete project: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return domain.ErrNotFound
	}

	// Cascade soft delete to project members
	if _, err := tx.Exec(ctx, `UPDATE project_members SET deleted_at = NOW() WHERE project_id = $1 AND deleted_at IS NULL`, id); err != nil {
		return fmt.Errorf("failed to cascade soft delete project members: %w", err)
	}

	// Cascade soft delete to sprints
	if _, err := tx.Exec(ctx, `UPDATE sprints SET deleted_at = NOW(), updated_at = NOW() WHERE project_id = $1 AND deleted_at IS NULL`, id); err != nil {
		return fmt.Errorf("failed to cascade soft delete sprints: %w", err)
	}

	// Cascade soft delete to issues
	if _, err := tx.Exec(ctx, `UPDATE issues SET deleted_at = NOW(), updated_at = NOW() WHERE project_id = $1 AND deleted_at IS NULL`, id); err != nil {
		return fmt.Errorf("failed to cascade soft delete issues: %w", err)
	}

	return tx.Commit(ctx)
}

func (r *projectRepository) IncrementIssueCounter(ctx context.Context, projectID uuid.UUID) (string, int64, error) {
	query := `
		UPDATE projects
		SET issue_counter = issue_counter + 1, updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING key, issue_counter
	`
	var (
		key     string
		counter int64
	)
	err := r.pool.QueryRow(ctx, query, projectID).Scan(&key, &counter)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", 0, domain.ErrNotFound
		}
		return "", 0, fmt.Errorf("failed to increment issue counter: %w", err)
	}
	return key, counter, nil
}

