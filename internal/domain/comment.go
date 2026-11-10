package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Comment struct {
	ID        uuid.UUID  `json:"id"`
	IssueID   uuid.UUID  `json:"issue_id"`
	UserID    uuid.UUID  `json:"user_id"`
	Content   string     `json:"content"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`

	User *User `json:"user,omitempty"`
}

type CreateCommentRequest struct {
	Content string `json:"content" validate:"required,min=1,max=10000"`
}

type UpdateCommentRequest struct {
	Content string `json:"content" validate:"required,min=1,max=10000"`
}

type CommentRepository interface {
	Create(ctx context.Context, comment *Comment) error
	GetByID(ctx context.Context, id uuid.UUID) (*Comment, error)
	ListByIssueID(ctx context.Context, issueID uuid.UUID) ([]Comment, error)
	Update(ctx context.Context, comment *Comment) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type CommentUsecase interface {
	CreateComment(ctx context.Context, userID, issueID uuid.UUID, req CreateCommentRequest) (*Comment, error)
	GetCommentsByIssueID(ctx context.Context, issueID uuid.UUID) ([]Comment, error)
	UpdateComment(ctx context.Context, userID, commentID uuid.UUID, req UpdateCommentRequest) (*Comment, error)
	DeleteComment(ctx context.Context, userID, commentID uuid.UUID) error
}
