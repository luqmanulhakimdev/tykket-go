package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

const (
	RoleOwner  = "OWNER"
	RoleAdmin  = "ADMIN"
	RoleMember = "MEMBER"
	RoleViewer = "VIEWER"

	PermProjectDelete = "project:delete"
	PermProjectUpdate = "project:update"
	PermMemberManage  = "member:manage"
	PermSprintManage  = "sprint:manage"
	PermIssueCreate   = "issue:create"
	PermIssueUpdate   = "issue:update"
	PermIssueDelete   = "issue:delete"
	PermReportView    = "report:view"
)

type Role struct {
	ID          uuid.UUID    `json:"id"`
	Name        string       `json:"name"`
	Description *string      `json:"description,omitempty"`
	CreatedAt   time.Time    `json:"created_at"`
	Permissions []Permission `json:"permissions,omitempty"`
}

type Permission struct {
	ID          uuid.UUID `json:"id"`
	Code        string    `json:"code"`
	Description *string   `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type ProjectMember struct {
	ID        uuid.UUID `json:"id"`
	ProjectID uuid.UUID `json:"project_id"`
	UserID    uuid.UUID `json:"user_id"`
	RoleID    uuid.UUID `json:"role_id"`
	JoinedAt  time.Time  `json:"joined_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
	User      *User      `json:"user,omitempty"`
	Role      *Role     `json:"role,omitempty"`
}

type RBACRepository interface {
	GetRoleByName(ctx context.Context, name string) (*Role, error)
	GetRoleByID(ctx context.Context, id uuid.UUID) (*Role, error)
	ListRoles(ctx context.Context) ([]Role, error)
	HasPermission(ctx context.Context, userID, projectID uuid.UUID, permissionCode string) (bool, error)
	GetUserMemberInProject(ctx context.Context, userID, projectID uuid.UUID) (*ProjectMember, error)
	ListProjectMembers(ctx context.Context, projectID uuid.UUID) ([]ProjectMember, error)
	AddProjectMember(ctx context.Context, member *ProjectMember) error
	UpdateProjectMemberRole(ctx context.Context, projectID, userID, roleID uuid.UUID) error
	RemoveProjectMember(ctx context.Context, projectID, userID uuid.UUID) error
}

