package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Project struct {
	ID           uuid.UUID       `json:"id"`
	Name         string          `json:"name"`
	Key          string          `json:"key"`
	Description  *string         `json:"description,omitempty"`
	OwnerID      uuid.UUID       `json:"owner_id"`
	IssueCounter int64           `json:"issue_counter"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
	DeletedAt    *time.Time      `json:"deleted_at,omitempty"`
	Owner        *User           `json:"owner,omitempty"`
	Members      []ProjectMember `json:"members,omitempty"`
}

type CreateProjectRequest struct {
	Name        string  `json:"name" validate:"required,min=2,max=150"`
	Key         string  `json:"key" validate:"required,min=2,max=10,alphanum"`
	Description *string `json:"description,omitempty" validate:"omitempty,max=1000"`
}

type UpdateProjectRequest struct {
	Name        string  `json:"name" validate:"required,min=2,max=150"`
	Description *string `json:"description,omitempty" validate:"omitempty,max=1000"`
}

type AddMemberRequest struct {
	Email    string `json:"email" validate:"required,email"`
	RoleName string `json:"role_name" validate:"required,oneof=OWNER ADMIN MEMBER VIEWER"`
}

type UpdateMemberRoleRequest struct {
	RoleName string `json:"role_name" validate:"required,oneof=ADMIN MEMBER VIEWER"`
}

type ProjectRepository interface {
	Create(ctx context.Context, project *Project) error
	GetByID(ctx context.Context, id uuid.UUID) (*Project, error)
	GetByKey(ctx context.Context, key string) (*Project, error)
	ListByUserID(ctx context.Context, userID uuid.UUID) ([]Project, error)
	Update(ctx context.Context, project *Project) error
	Delete(ctx context.Context, id uuid.UUID) error
	IncrementIssueCounter(ctx context.Context, projectID uuid.UUID) (string, int64, error)
}

type ProjectUsecase interface {
	CreateProject(ctx context.Context, userID uuid.UUID, req CreateProjectRequest) (*Project, error)
	GetProjectByID(ctx context.Context, userID, projectID uuid.UUID) (*Project, error)
	ListProjects(ctx context.Context, userID uuid.UUID) ([]Project, error)
	UpdateProject(ctx context.Context, userID, projectID uuid.UUID, req UpdateProjectRequest) (*Project, error)
	DeleteProject(ctx context.Context, userID, projectID uuid.UUID) error

	// Member management
	ListMembers(ctx context.Context, projectID uuid.UUID) ([]ProjectMember, error)
	AddMember(ctx context.Context, projectID uuid.UUID, req AddMemberRequest) (*ProjectMember, error)
	UpdateMemberRole(ctx context.Context, projectID, targetUserID uuid.UUID, req UpdateMemberRoleRequest) (*ProjectMember, error)
	RemoveMember(ctx context.Context, projectID, targetUserID uuid.UUID) error
}

