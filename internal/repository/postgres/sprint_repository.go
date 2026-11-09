package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/luqmanulhakimdev/tykket-go/internal/domain"
)

type sprintRepository struct {
	pool *pgxpool.Pool
}

func NewSprintRepository(pool *pgxpool.Pool) domain.SprintRepository {
	return &sprintRepository{pool: pool}
}

func (r *sprintRepository) Create(ctx context.Context, s *domain.Sprint) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	if s.Status == "" {
		s.Status = domain.SprintStatusPlanned
	}

	query := `
		INSERT INTO sprints (id, project_id, name, goal, start_date, end_date, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
		RETURNING created_at, updated_at
	`
	err := r.pool.QueryRow(ctx, query,
		s.ID,
		s.ProjectID,
		s.Name,
		s.Goal,
		s.StartDate,
		s.EndDate,
		s.Status,
	).Scan(&s.CreatedAt, &s.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create sprint: %w", err)
	}
	return nil
}

func (r *sprintRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Sprint, error) {
	query := `
		SELECT id, project_id, name, goal, start_date, end_date, status, created_at, updated_at, deleted_at
		FROM sprints
		WHERE id = $1 AND deleted_at IS NULL
	`
	var s domain.Sprint
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&s.ID,
		&s.ProjectID,
		&s.Name,
		&s.Goal,
		&s.StartDate,
		&s.EndDate,
		&s.Status,
		&s.CreatedAt,
		&s.UpdatedAt,
		&s.DeletedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get sprint by id: %w", err)
	}
	return &s, nil
}

func (r *sprintRepository) GetActiveByProjectID(ctx context.Context, projectID uuid.UUID) (*domain.Sprint, error) {
	query := `
		SELECT id, project_id, name, goal, start_date, end_date, status, created_at, updated_at, deleted_at
		FROM sprints
		WHERE project_id = $1 AND status = 'ACTIVE' AND deleted_at IS NULL
		LIMIT 1
	`
	var s domain.Sprint
	err := r.pool.QueryRow(ctx, query, projectID).Scan(
		&s.ID,
		&s.ProjectID,
		&s.Name,
		&s.Goal,
		&s.StartDate,
		&s.EndDate,
		&s.Status,
		&s.CreatedAt,
		&s.UpdatedAt,
		&s.DeletedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // Return nil when no active sprint exists
		}
		return nil, fmt.Errorf("failed to get active sprint: %w", err)
	}
	return &s, nil
}

func (r *sprintRepository) ListByProjectID(ctx context.Context, projectID uuid.UUID) ([]domain.Sprint, error) {
	query := `
		SELECT id, project_id, name, goal, start_date, end_date, status, created_at, updated_at, deleted_at
		FROM sprints
		WHERE project_id = $1 AND deleted_at IS NULL
		ORDER BY 
			CASE status
				WHEN 'ACTIVE' THEN 1
				WHEN 'PLANNED' THEN 2
				WHEN 'COMPLETED' THEN 3
				ELSE 4
			END,
			created_at DESC
	`
	rows, err := r.pool.Query(ctx, query, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list sprints: %w", err)
	}
	defer rows.Close()

	var sprints []domain.Sprint
	for rows.Next() {
		var s domain.Sprint
		if err := rows.Scan(
			&s.ID,
			&s.ProjectID,
			&s.Name,
			&s.Goal,
			&s.StartDate,
			&s.EndDate,
			&s.Status,
			&s.CreatedAt,
			&s.UpdatedAt,
			&s.DeletedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan sprint: %w", err)
		}
		sprints = append(sprints, s)
	}
	return sprints, nil
}

func (r *sprintRepository) Update(ctx context.Context, s *domain.Sprint) error {
	query := `
		UPDATE sprints
		SET name = $1, goal = $2, start_date = $3, end_date = $4, status = $5, updated_at = NOW()
		WHERE id = $6 AND deleted_at IS NULL
		RETURNING updated_at
	`
	err := r.pool.QueryRow(ctx, query,
		s.Name,
		s.Goal,
		s.StartDate,
		s.EndDate,
		s.Status,
		s.ID,
	).Scan(&s.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrNotFound
		}
		return fmt.Errorf("failed to update sprint: %w", err)
	}
	return nil
}

func (r *sprintRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE sprints
		SET deleted_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`
	ct, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete sprint: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

