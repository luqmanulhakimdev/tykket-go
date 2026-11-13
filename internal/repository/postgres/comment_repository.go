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

type commentRepository struct {
	pool *pgxpool.Pool
}

func NewCommentRepository(pool *pgxpool.Pool) domain.CommentRepository {
	return &commentRepository{pool: pool}
}

func (r *commentRepository) Create(ctx context.Context, c *domain.Comment) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}

	query := `
		INSERT INTO comments (id, issue_id, user_id, parent_id, content, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		RETURNING created_at, updated_at
	`

	err := r.pool.QueryRow(ctx, query,
		c.ID,
		c.IssueID,
		c.UserID,
		c.ParentID,
		c.Content,
	).Scan(&c.CreatedAt, &c.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create comment: %w", err)
	}

	return nil
}

func (r *commentRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Comment, error) {
	query := `
		SELECT 
			c.id, c.issue_id, c.user_id, c.parent_id, c.content, c.created_at, c.updated_at, c.deleted_at,
			u.id, u.email, u.name, u.avatar_url
		FROM comments c
		INNER JOIN users u ON u.id = c.user_id
		WHERE c.id = $1 AND c.deleted_at IS NULL
	`

	var comment domain.Comment
	var user domain.User

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&comment.ID,
		&comment.IssueID,
		&comment.UserID,
		&comment.ParentID,
		&comment.Content,
		&comment.CreatedAt,
		&comment.UpdatedAt,
		&comment.DeletedAt,
		&user.ID,
		&user.Email,
		&user.Name,
		&user.AvatarURL,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get comment: %w", err)
	}

	comment.User = &user
	return &comment, nil
}

func (r *commentRepository) ListByIssueID(ctx context.Context, issueID uuid.UUID) ([]domain.Comment, error) {
	query := `
		SELECT 
			c.id, c.issue_id, c.user_id, c.parent_id, c.content, c.created_at, c.updated_at, c.deleted_at,
			u.id, u.email, u.name, u.avatar_url
		FROM comments c
		INNER JOIN users u ON u.id = c.user_id
		WHERE c.issue_id = $1 AND c.deleted_at IS NULL
		ORDER BY c.created_at ASC
	`

	rows, err := r.pool.Query(ctx, query, issueID)
	if err != nil {
		return nil, fmt.Errorf("failed to list comments: %w", err)
	}
	defer rows.Close()

	var allComments []domain.Comment
	for rows.Next() {
		var c domain.Comment
		var u domain.User
		if err := rows.Scan(
			&c.ID,
			&c.IssueID,
			&c.UserID,
			&c.ParentID,
			&c.Content,
			&c.CreatedAt,
			&c.UpdatedAt,
			&c.DeletedAt,
			&u.ID,
			&u.Email,
			&u.Name,
			&u.AvatarURL,
		); err != nil {
			return nil, fmt.Errorf("failed to scan comment row: %w", err)
		}
		c.User = &u
		c.Replies = []domain.Comment{}
		allComments = append(allComments, c)
	}

	// Build threaded comment structure (root comments with nested replies)
	commentMap := make(map[uuid.UUID]*domain.Comment)
	for i := range allComments {
		commentMap[allComments[i].ID] = &allComments[i]
	}

	// Helper to find the root comment for any ancestor chain
	findRootComment := func(c *domain.Comment) *domain.Comment {
		curr := c
		visited := make(map[uuid.UUID]bool)
		for curr.ParentID != nil {
			if visited[curr.ID] {
				break
			}
			visited[curr.ID] = true
			parent, exists := commentMap[*curr.ParentID]
			if !exists {
				break
			}
			curr = parent
		}
		return curr
	}

	// First pass: attach all replies to their thread root
	for i := range allComments {
		if allComments[i].ParentID != nil {
			root := findRootComment(&allComments[i])
			if root != nil && root.ID != allComments[i].ID {
				root.Replies = append(root.Replies, allComments[i])
			}
		}
	}

	// Second pass: collect root comments with their replies populated
	var rootComments []domain.Comment
	for i := range allComments {
		if allComments[i].ParentID == nil {
			rootComments = append(rootComments, allComments[i])
		} else if _, exists := commentMap[*allComments[i].ParentID]; !exists {
			// Orphaned comment fallback
			rootComments = append(rootComments, allComments[i])
		}
	}

	if rootComments == nil {
		rootComments = []domain.Comment{}
	}

	return rootComments, nil
}

func (r *commentRepository) Update(ctx context.Context, c *domain.Comment) error {
	query := `
		UPDATE comments
		SET content = $1, updated_at = NOW()
		WHERE id = $2 AND deleted_at IS NULL
		RETURNING updated_at
	`

	err := r.pool.QueryRow(ctx, query, c.Content, c.ID).Scan(&c.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrNotFound
		}
		return fmt.Errorf("failed to update comment: %w", err)
	}

	return nil
}

func (r *commentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE comments
		SET deleted_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`

	cmd, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete comment: %w", err)
	}

	if cmd.RowsAffected() == 0 {
		return domain.ErrNotFound
	}

	return nil
}

