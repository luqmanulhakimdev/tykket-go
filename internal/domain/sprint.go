package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type SprintStatus string

const (
	SprintStatusPlanned   SprintStatus = "PLANNED"
	SprintStatusActive    SprintStatus = "ACTIVE"
	SprintStatusCompleted SprintStatus = "COMPLETED"
)

type Sprint struct {
	ID        uuid.UUID    `json:"id"`
	ProjectID uuid.UUID    `json:"project_id"`
	Name      string       `json:"name"`
	Goal      *string      `json:"goal,omitempty"`
	StartDate *time.Time   `json:"start_date,omitempty"`
	EndDate   *time.Time   `json:"end_date,omitempty"`
	Status    SprintStatus `json:"status"`
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
	DeletedAt *time.Time   `json:"deleted_at,omitempty"`
}

type CreateSprintRequest struct {
	Name      string     `json:"name" validate:"required,min=2,max=150"`
	Goal      *string    `json:"goal,omitempty" validate:"omitempty,max=1000"`
	StartDate *time.Time `json:"start_date,omitempty"`
	EndDate   *time.Time `json:"end_date,omitempty"`
}

type UpdateSprintRequest struct {
	Name      string     `json:"name" validate:"required,min=2,max=150"`
	Goal      *string    `json:"goal,omitempty" validate:"omitempty,max=1000"`
	StartDate *time.Time `json:"start_date,omitempty"`
	EndDate   *time.Time `json:"end_date,omitempty"`
}

type StartSprintRequest struct {
	StartDate *time.Time `json:"start_date,omitempty"`
	EndDate   *time.Time `json:"end_date,omitempty"`
}

type CompleteSprintRequest struct {
	MoveIncompleteToSprintID *uuid.UUID `json:"move_incomplete_to_sprint_id,omitempty"`
}

type BulkMoveSprintIssuesRequest struct {
	IssueIDs []uuid.UUID `json:"issue_ids" validate:"required,min=1"`
	SprintID *uuid.UUID  `json:"sprint_id"` // null to move to Backlog
}

type SprintRepository interface {
	Create(ctx context.Context, sprint *Sprint) error
	GetByID(ctx context.Context, id uuid.UUID) (*Sprint, error)
	GetActiveByProjectID(ctx context.Context, projectID uuid.UUID) (*Sprint, error)
	ListByProjectID(ctx context.Context, projectID uuid.UUID) ([]Sprint, error)
	Update(ctx context.Context, sprint *Sprint) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type SprintUsecase interface {
	CreateSprint(ctx context.Context, projectID uuid.UUID, req CreateSprintRequest) (*Sprint, error)
	GetSprintByID(ctx context.Context, projectID, sprintID uuid.UUID) (*Sprint, error)
	ListSprints(ctx context.Context, projectID uuid.UUID) ([]Sprint, error)
	UpdateSprint(ctx context.Context, projectID, sprintID uuid.UUID, req UpdateSprintRequest) (*Sprint, error)
	StartSprint(ctx context.Context, projectID, sprintID uuid.UUID, req StartSprintRequest) (*Sprint, error)
	CompleteSprint(ctx context.Context, projectID, sprintID uuid.UUID, req CompleteSprintRequest) (*Sprint, error)
	DeleteSprint(ctx context.Context, projectID, sprintID uuid.UUID) error
	BulkMoveIssues(ctx context.Context, projectID uuid.UUID, req BulkMoveSprintIssuesRequest) error
}

