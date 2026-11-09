package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/luqmanulhakimdev/tykket-go/internal/domain"
)

type issueRepository struct {
	pool *pgxpool.Pool
}

func NewIssueRepository(pool *pgxpool.Pool) domain.IssueRepository {
	return &issueRepository{pool: pool}
}

func (r *issueRepository) Create(ctx context.Context, i *domain.Issue) error {
	if i.ID == uuid.Nil {
		i.ID = uuid.New()
	}
	if i.Status == "" {
		i.Status = domain.IssueStatusTodo
	}
	if i.Priority == "" {
		i.Priority = domain.IssuePriorityMedium
	}
	if i.Position == 0 {
		i.Position = 65535.0
	}

	query := `
		INSERT INTO issues (
			id, project_id, sprint_id, parent_id, key, issue_number,
			type, status, priority, title, description,
			reporter_id, assignee_id, story_points, position,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10, $11,
			$12, $13, $14, $15,
			NOW(), NOW()
		)
		RETURNING created_at, updated_at
	`

	err := r.pool.QueryRow(ctx, query,
		i.ID,
		i.ProjectID,
		i.SprintID,
		i.ParentID,
		i.Key,
		i.IssueNumber,
		i.Type,
		i.Status,
		i.Priority,
		i.Title,
		i.Description,
		i.ReporterID,
		i.AssigneeID,
		i.StoryPoints,
		i.Position,
	).Scan(&i.CreatedAt, &i.UpdatedAt)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.ErrConflict
		}
		return fmt.Errorf("failed to create issue: %w", err)
	}

	return nil
}

func (r *issueRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Issue, error) {
	query := `
		SELECT 
			i.id, i.project_id, i.sprint_id, i.parent_id, i.key, i.issue_number,
			i.type, i.status, i.priority, i.title, i.description,
			i.reporter_id, i.assignee_id, i.story_points, i.position,
			i.created_at, i.updated_at, i.deleted_at,
			ru.id, ru.email, ru.name, ru.avatar_url,
			au.id, au.email, au.name, au.avatar_url
		FROM issues i
		JOIN users ru ON i.reporter_id = ru.id AND ru.deleted_at IS NULL
		LEFT JOIN users au ON i.assignee_id = au.id AND au.deleted_at IS NULL
		WHERE i.id = $1 AND i.deleted_at IS NULL
	`
	var (
		i      domain.Issue
		ru     domain.User
		auID   *uuid.UUID
		auMail *string
		auName *string
		auAv   *string
	)

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&i.ID,
		&i.ProjectID,
		&i.SprintID,
		&i.ParentID,
		&i.Key,
		&i.IssueNumber,
		&i.Type,
		&i.Status,
		&i.Priority,
		&i.Title,
		&i.Description,
		&i.ReporterID,
		&i.AssigneeID,
		&i.StoryPoints,
		&i.Position,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.DeletedAt,
		&ru.ID,
		&ru.Email,
		&ru.Name,
		&ru.AvatarURL,
		&auID,
		&auMail,
		&auName,
		&auAv,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get issue by id: %w", err)
	}

	i.Reporter = &ru
	if auID != nil {
		i.Assignee = &domain.User{
			ID:        *auID,
			Email:     *auMail,
			Name:      *auName,
			AvatarURL: auAv,
		}
	}

	// Fetch subtasks if any
	subtasks, _, err := r.List(ctx, domain.IssueFilter{
		ProjectID: i.ProjectID,
		ParentID:  &i.ID,
	})
	if err == nil {
		i.Subtasks = subtasks
	}

	return &i, nil
}

func (r *issueRepository) GetByKey(ctx context.Context, key string) (*domain.Issue, error) {
	query := `
		SELECT 
			i.id, i.project_id, i.sprint_id, i.parent_id, i.key, i.issue_number,
			i.type, i.status, i.priority, i.title, i.description,
			i.reporter_id, i.assignee_id, i.story_points, i.position,
			i.created_at, i.updated_at, i.deleted_at,
			ru.id, ru.email, ru.name, ru.avatar_url,
			au.id, au.email, au.name, au.avatar_url
		FROM issues i
		JOIN users ru ON i.reporter_id = ru.id AND ru.deleted_at IS NULL
		LEFT JOIN users au ON i.assignee_id = au.id AND au.deleted_at IS NULL
		WHERE UPPER(i.key) = UPPER($1) AND i.deleted_at IS NULL
	`
	var (
		i      domain.Issue
		ru     domain.User
		auID   *uuid.UUID
		auMail *string
		auName *string
		auAv   *string
	)

	err := r.pool.QueryRow(ctx, query, key).Scan(
		&i.ID,
		&i.ProjectID,
		&i.SprintID,
		&i.ParentID,
		&i.Key,
		&i.IssueNumber,
		&i.Type,
		&i.Status,
		&i.Priority,
		&i.Title,
		&i.Description,
		&i.ReporterID,
		&i.AssigneeID,
		&i.StoryPoints,
		&i.Position,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.DeletedAt,
		&ru.ID,
		&ru.Email,
		&ru.Name,
		&ru.AvatarURL,
		&auID,
		&auMail,
		&auName,
		&auAv,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get issue by key: %w", err)
	}

	i.Reporter = &ru
	if auID != nil {
		i.Assignee = &domain.User{
			ID:        *auID,
			Email:     *auMail,
			Name:      *auName,
			AvatarURL: auAv,
		}
	}

	return &i, nil
}

func (r *issueRepository) List(ctx context.Context, filter domain.IssueFilter) ([]domain.Issue, int64, error) {
	var (
		conditions = []string{"i.deleted_at IS NULL"}
		args       []any
		argIdx     = 1
	)

	if filter.ProjectID != uuid.Nil {
		conditions = append(conditions, fmt.Sprintf("i.project_id = $%d", argIdx))
		args = append(args, filter.ProjectID)
		argIdx++
	}

	if filter.InBacklog != nil && *filter.InBacklog {
		conditions = append(conditions, "i.sprint_id IS NULL")
	} else if filter.SprintID != nil {
		conditions = append(conditions, fmt.Sprintf("i.sprint_id = $%d", argIdx))
		args = append(args, *filter.SprintID)
		argIdx++
	}

	if filter.ParentID != nil {
		conditions = append(conditions, fmt.Sprintf("i.parent_id = $%d", argIdx))
		args = append(args, *filter.ParentID)
		argIdx++
	}

	if filter.Type != nil {
		conditions = append(conditions, fmt.Sprintf("i.type = $%d", argIdx))
		args = append(args, *filter.Type)
		argIdx++
	}

	if filter.Status != nil {
		conditions = append(conditions, fmt.Sprintf("i.status = $%d", argIdx))
		args = append(args, *filter.Status)
		argIdx++
	}

	if filter.AssigneeID != nil {
		conditions = append(conditions, fmt.Sprintf("i.assignee_id = $%d", argIdx))
		args = append(args, *filter.AssigneeID)
		argIdx++
	}

	if filter.ReporterID != nil {
		conditions = append(conditions, fmt.Sprintf("i.reporter_id = $%d", argIdx))
		args = append(args, *filter.ReporterID)
		argIdx++
	}

	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(i.title ILIKE $%d OR i.key ILIKE $%d OR i.description ILIKE $%d)", argIdx, argIdx, argIdx))
		args = append(args, "%"+filter.Search+"%")
		argIdx++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Count total query
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM issues i %s", whereClause)
	var total int64
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count issues: %w", err)
	}

	// Ordering
	orderBy := "i.position ASC"
	if filter.SortBy != "" {
		order := "ASC"
		if strings.EqualFold(filter.SortOrder, "DESC") {
			order = "DESC"
		}
		switch filter.SortBy {
		case "created_at":
			orderBy = fmt.Sprintf("i.created_at %s", order)
		case "updated_at":
			orderBy = fmt.Sprintf("i.updated_at %s", order)
		case "priority":
			orderBy = fmt.Sprintf("i.priority %s", order)
		case "status":
			orderBy = fmt.Sprintf("i.status %s", order)
		default:
			orderBy = fmt.Sprintf("i.position %s", order)
		}
	}

	limitClause := ""
	if filter.Limit > 0 {
		limitClause = fmt.Sprintf("LIMIT %d OFFSET %d", filter.Limit, filter.Offset)
	}

	query := fmt.Sprintf(`
		SELECT 
			i.id, i.project_id, i.sprint_id, i.parent_id, i.key, i.issue_number,
			i.type, i.status, i.priority, i.title, i.description,
			i.reporter_id, i.assignee_id, i.story_points, i.position,
			i.created_at, i.updated_at, i.deleted_at,
			ru.id, ru.email, ru.name, ru.avatar_url,
			au.id, au.email, au.name, au.avatar_url
		FROM issues i
		JOIN users ru ON i.reporter_id = ru.id AND ru.deleted_at IS NULL
		LEFT JOIN users au ON i.assignee_id = au.id AND au.deleted_at IS NULL
		%s
		ORDER BY %s
		%s
	`, whereClause, orderBy, limitClause)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list issues: %w", err)
	}
	defer rows.Close()

	var issues []domain.Issue
	for rows.Next() {
		var (
			i      domain.Issue
			ru     domain.User
			auID   *uuid.UUID
			auMail *string
			auName *string
			auAv   *string
		)

		if err := rows.Scan(
			&i.ID,
			&i.ProjectID,
			&i.SprintID,
			&i.ParentID,
			&i.Key,
			&i.IssueNumber,
			&i.Type,
			&i.Status,
			&i.Priority,
			&i.Title,
			&i.Description,
			&i.ReporterID,
			&i.AssigneeID,
			&i.StoryPoints,
			&i.Position,
			&i.CreatedAt,
			&i.UpdatedAt,
			&i.DeletedAt,
			&ru.ID,
			&ru.Email,
			&ru.Name,
			&ru.AvatarURL,
			&auID,
			&auMail,
			&auName,
			&auAv,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan issue: %w", err)
		}

		i.Reporter = &ru
		if auID != nil {
			i.Assignee = &domain.User{
				ID:        *auID,
				Email:     *auMail,
				Name:      *auName,
				AvatarURL: auAv,
			}
		}

		issues = append(issues, i)
	}

	return issues, total, nil
}

func (r *issueRepository) Update(ctx context.Context, i *domain.Issue) error {
	query := `
		UPDATE issues
		SET 
			sprint_id = $1,
			parent_id = $2,
			type = $3,
			priority = $4,
			title = $5,
			description = $6,
			assignee_id = $7,
			story_points = $8,
			updated_at = NOW()
		WHERE id = $9 AND deleted_at IS NULL
		RETURNING updated_at
	`
	err := r.pool.QueryRow(ctx, query,
		i.SprintID,
		i.ParentID,
		i.Type,
		i.Priority,
		i.Title,
		i.Description,
		i.AssigneeID,
		i.StoryPoints,
		i.ID,
	).Scan(&i.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrNotFound
		}
		return fmt.Errorf("failed to update issue: %w", err)
	}
	return nil
}

func (r *issueRepository) UpdatePositionAndStatus(ctx context.Context, id uuid.UUID, status *domain.IssueStatus, position float64, sprintID *uuid.UUID) error {
	setClauses := []string{"position = $1", "updated_at = NOW()"}
	args := []any{position, id}
	argIdx := 3

	if status != nil {
		setClauses = append(setClauses, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, *status)
		argIdx++
	}

	if sprintID != nil {
		setClauses = append(setClauses, fmt.Sprintf("sprint_id = $%d", argIdx))
		args = append(args, *sprintID)
		argIdx++
	}

	query := fmt.Sprintf(`
		UPDATE issues
		SET %s
		WHERE id = $2 AND deleted_at IS NULL
	`, strings.Join(setClauses, ", "))

	ct, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update issue position/status: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *issueRepository) Delete(ctx context.Context, id uuid.UUID) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		UPDATE issues
		SET deleted_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`
	ct, err := tx.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to soft delete issue: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return domain.ErrNotFound
	}

	// Cascade soft delete to subtasks
	if _, err := tx.Exec(ctx, `UPDATE issues SET deleted_at = NOW(), updated_at = NOW() WHERE parent_id = $1 AND deleted_at IS NULL`, id); err != nil {
		return fmt.Errorf("failed to cascade soft delete subtasks: %w", err)
	}

	return tx.Commit(ctx)
}

func (r *issueRepository) MoveIncompleteIssues(ctx context.Context, fromSprintID uuid.UUID, toSprintID *uuid.UUID) error {
	query := `
		UPDATE issues
		SET sprint_id = $1, updated_at = NOW()
		WHERE sprint_id = $2 AND status != 'DONE' AND deleted_at IS NULL
	`
	_, err := r.pool.Exec(ctx, query, toSprintID, fromSprintID)
	if err != nil {
		return fmt.Errorf("failed to move incomplete issues: %w", err)
	}
	return nil
}

func (r *issueRepository) BulkMoveIssues(ctx context.Context, issueIDs []uuid.UUID, sprintID *uuid.UUID) error {
	query := `
		UPDATE issues
		SET sprint_id = $1, updated_at = NOW()
		WHERE id = ANY($2) AND deleted_at IS NULL
	`
	_, err := r.pool.Exec(ctx, query, sprintID, issueIDs)
	if err != nil {
		return fmt.Errorf("failed to bulk move issues: %w", err)
	}
	return nil
}

