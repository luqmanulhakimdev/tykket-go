package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type IssueType string

const (
	IssueTypeEpic    IssueType = "EPIC"
	IssueTypeStory   IssueType = "STORY"
	IssueTypeTask    IssueType = "TASK"
	IssueTypeBug     IssueType = "BUG"
	IssueTypeSubtask IssueType = "SUBTASK"
)

type IssueStatus string

const (
	IssueStatusTodo       IssueStatus = "TODO"
	IssueStatusInProgress IssueStatus = "IN_PROGRESS"
	IssueStatusInReview   IssueStatus = "IN_REVIEW"
	IssueStatusDone       IssueStatus = "DONE"
)

type IssuePriority string

const (
	IssuePriorityLow    IssuePriority = "LOW"
	IssuePriorityMedium IssuePriority = "MEDIUM"
	IssuePriorityHigh   IssuePriority = "HIGH"
	IssuePriorityUrgent IssuePriority = "URGENT"
)

type Issue struct {
	ID          uuid.UUID     `json:"id"`
	ProjectID   uuid.UUID     `json:"project_id"`
	SprintID    *uuid.UUID    `json:"sprint_id"`
	ParentID    *uuid.UUID    `json:"parent_id"`
	Key         string        `json:"key"`
	IssueNumber int64         `json:"issue_number"`
	Type        IssueType     `json:"type"`
	Status      IssueStatus   `json:"status"`
	Priority    IssuePriority `json:"priority"`
	Title       string        `json:"title"`
	Description *string       `json:"description,omitempty"`
	ReporterID  uuid.UUID     `json:"reporter_id"`
	AssigneeID  *uuid.UUID    `json:"assignee_id,omitempty"`
	StoryPoints *int          `json:"story_points,omitempty"`
	Position    float64       `json:"position"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
	DeletedAt   *time.Time    `json:"deleted_at,omitempty"`

	Reporter *User   `json:"reporter,omitempty"`
	Assignee *User   `json:"assignee,omitempty"`
	Parent   *Issue  `json:"parent,omitempty"`
	Subtasks []Issue `json:"subtasks,omitempty"`
}

type CreateIssueRequest struct {
	SprintID    *uuid.UUID    `json:"sprint_id,omitempty"`
	ParentID    *uuid.UUID    `json:"parent_id,omitempty"`
	Type        IssueType     `json:"type" validate:"required,oneof=EPIC STORY TASK BUG SUBTASK"`
	Status      *IssueStatus  `json:"status,omitempty" validate:"omitempty,oneof=TODO IN_PROGRESS IN_REVIEW DONE"`
	Priority    IssuePriority `json:"priority" validate:"required,oneof=LOW MEDIUM HIGH URGENT"`
	Title       string        `json:"title" validate:"required,min=2,max=255"`
	Description *string       `json:"description,omitempty"`
	AssigneeID  *uuid.UUID    `json:"assignee_id,omitempty"`
	StoryPoints *int          `json:"story_points,omitempty" validate:"omitempty,min=0,max=100"`
	Position    *float64      `json:"position,omitempty"`
}

type UpdateIssueRequest struct {
	SprintID    *uuid.UUID     `json:"sprint_id,omitempty"`
	ParentID    *uuid.UUID     `json:"parent_id,omitempty"`
	Type        *IssueType     `json:"type,omitempty" validate:"omitempty,oneof=EPIC STORY TASK BUG SUBTASK"`
	Priority    *IssuePriority `json:"priority,omitempty" validate:"omitempty,oneof=LOW MEDIUM HIGH URGENT"`
	Title       *string        `json:"title,omitempty" validate:"omitempty,min=2,max=255"`
	Description *string        `json:"description,omitempty"`
	AssigneeID  *uuid.UUID     `json:"assignee_id,omitempty"`
	StoryPoints *int           `json:"story_points,omitempty" validate:"omitempty,min=0,max=100"`
}

type MoveIssueRequest struct {
	Status   *IssueStatus `json:"status,omitempty" validate:"omitempty,oneof=TODO IN_PROGRESS IN_REVIEW DONE"`
	Position float64      `json:"position" validate:"required"`
	SprintID *uuid.UUID   `json:"sprint_id,omitempty"` // allow moving between sprint/backlog during reorder
}

type IssueFilter struct {
	ProjectID  uuid.UUID
	SprintID   *uuid.UUID
	InBacklog  *bool
	ParentID   *uuid.UUID
	Type       *IssueType
	Status     *IssueStatus
	AssigneeID *uuid.UUID
	ReporterID *uuid.UUID
	Search     string
	SortBy     string
	SortOrder  string
	Limit      int
	Offset     int
}

type IssueRepository interface {
	Create(ctx context.Context, issue *Issue) error
	GetByID(ctx context.Context, id uuid.UUID) (*Issue, error)
	GetByKey(ctx context.Context, key string) (*Issue, error)
	List(ctx context.Context, filter IssueFilter) ([]Issue, int64, error)
	Update(ctx context.Context, issue *Issue) error
	UpdatePositionAndStatus(ctx context.Context, id uuid.UUID, status *IssueStatus, position float64, sprintID *uuid.UUID) error
	Delete(ctx context.Context, id uuid.UUID) error
	MoveIncompleteIssues(ctx context.Context, fromSprintID uuid.UUID, toSprintID *uuid.UUID) error
	BulkMoveIssues(ctx context.Context, issueIDs []uuid.UUID, sprintID *uuid.UUID) error
}

type IssueUsecase interface {
	CreateIssue(ctx context.Context, reporterID, projectID uuid.UUID, req CreateIssueRequest) (*Issue, error)
	GetIssueByID(ctx context.Context, projectID, issueID uuid.UUID) (*Issue, error)
	GetIssueByKey(ctx context.Context, key string) (*Issue, error)
	ListIssues(ctx context.Context, filter IssueFilter) ([]Issue, int64, error)
	UpdateIssue(ctx context.Context, projectID, issueID uuid.UUID, req UpdateIssueRequest) (*Issue, error)
	MoveIssue(ctx context.Context, projectID, issueID uuid.UUID, req MoveIssueRequest) (*Issue, error)
	DeleteIssue(ctx context.Context, projectID, issueID uuid.UUID) error
}

