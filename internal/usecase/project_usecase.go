package usecase

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/luqmanulhakimdev/tykket-go/internal/domain"
)

var keyRegex = regexp.MustCompile(`^[A-Z0-9]{2,10}$`)

type projectUsecase struct {
	projectRepo domain.ProjectRepository
	rbacRepo    domain.RBACRepository
	userRepo    domain.UserRepository
}

func NewProjectUsecase(
	projectRepo domain.ProjectRepository,
	rbacRepo domain.RBACRepository,
	userRepo domain.UserRepository,
) domain.ProjectUsecase {
	return &projectUsecase{
		projectRepo: projectRepo,
		rbacRepo:    rbacRepo,
		userRepo:    userRepo,
	}
}

func (u *projectUsecase) CreateProject(ctx context.Context, userID uuid.UUID, req domain.CreateProjectRequest) (*domain.Project, error) {
	key := strings.ToUpper(strings.TrimSpace(req.Key))
	if !keyRegex.MatchString(key) {
		return nil, fmt.Errorf("%w: project key must be 2-10 alphanumeric characters (uppercase)", domain.ErrInvalidInput)
	}

	// Verify key uniqueness
	existing, err := u.projectRepo.GetByKey(ctx, key)
	if err == nil && existing != nil {
		return nil, fmt.Errorf("%w: project key '%s' is already in use", domain.ErrConflict, key)
	}
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return nil, err
	}

	project := &domain.Project{
		ID:          uuid.New(),
		Name:        strings.TrimSpace(req.Name),
		Key:         key,
		Description: req.Description,
		OwnerID:     userID,
	}

	if err := u.projectRepo.Create(ctx, project); err != nil {
		return nil, err
	}

	// Automatically assign owner as member with OWNER role
	ownerRole, err := u.rbacRepo.GetRoleByName(ctx, domain.RoleOwner)
	if err == nil && ownerRole != nil {
		_ = u.rbacRepo.AddProjectMember(ctx, &domain.ProjectMember{
			ID:        uuid.New(),
			ProjectID: project.ID,
			UserID:    userID,
			RoleID:    ownerRole.ID,
		})
	}

	return project, nil
}

func (u *projectUsecase) GetProjectByID(ctx context.Context, userID, projectID uuid.UUID) (*domain.Project, error) {
	project, err := u.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		return nil, err
	}

	members, err := u.rbacRepo.ListProjectMembers(ctx, projectID)
	if err == nil {
		project.Members = members
	}

	return project, nil
}

func (u *projectUsecase) ListProjects(ctx context.Context, userID uuid.UUID) ([]domain.Project, error) {
	return u.projectRepo.ListByUserID(ctx, userID)
}

func (u *projectUsecase) UpdateProject(ctx context.Context, userID, projectID uuid.UUID, req domain.UpdateProjectRequest) (*domain.Project, error) {
	project, err := u.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		return nil, err
	}

	project.Name = strings.TrimSpace(req.Name)
	project.Description = req.Description

	if err := u.projectRepo.Update(ctx, project); err != nil {
		return nil, err
	}

	return project, nil
}

func (u *projectUsecase) DeleteProject(ctx context.Context, userID, projectID uuid.UUID) error {
	project, err := u.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		return err
	}

	if project.OwnerID != userID {
		return domain.ErrForbidden
	}

	return u.projectRepo.Delete(ctx, projectID)
}

func (u *projectUsecase) ListMembers(ctx context.Context, projectID uuid.UUID) ([]domain.ProjectMember, error) {
	return u.rbacRepo.ListProjectMembers(ctx, projectID)
}

func (u *projectUsecase) AddMember(ctx context.Context, projectID uuid.UUID, req domain.AddMemberRequest) (*domain.ProjectMember, error) {
	targetEmail := strings.TrimSpace(strings.ToLower(req.Email))

	user, err := u.userRepo.GetByEmail(ctx, targetEmail)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, fmt.Errorf("%w: user with email %s not found", domain.ErrNotFound, targetEmail)
		}
		return nil, err
	}

	role, err := u.rbacRepo.GetRoleByName(ctx, req.RoleName)
	if err != nil {
		return nil, fmt.Errorf("%w: role %s not found", domain.ErrNotFound, req.RoleName)
	}

	member := &domain.ProjectMember{
		ID:        uuid.New(),
		ProjectID: projectID,
		UserID:    user.ID,
		RoleID:    role.ID,
	}

	if err := u.rbacRepo.AddProjectMember(ctx, member); err != nil {
		return nil, err
	}

	member.User = user
	member.Role = role
	return member, nil
}

func (u *projectUsecase) UpdateMemberRole(ctx context.Context, projectID, targetUserID uuid.UUID, req domain.UpdateMemberRoleRequest) (*domain.ProjectMember, error) {
	role, err := u.rbacRepo.GetRoleByName(ctx, req.RoleName)
	if err != nil {
		return nil, fmt.Errorf("%w: role %s not found", domain.ErrNotFound, req.RoleName)
	}

	if err := u.rbacRepo.UpdateProjectMemberRole(ctx, projectID, targetUserID, role.ID); err != nil {
		return nil, err
	}

	return u.rbacRepo.GetUserMemberInProject(ctx, targetUserID, projectID)
}

func (u *projectUsecase) RemoveMember(ctx context.Context, projectID, targetUserID uuid.UUID) error {
	project, err := u.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		return err
	}

	// Cannot remove project owner
	if project.OwnerID == targetUserID {
		return fmt.Errorf("%w: cannot remove project owner from project members", domain.ErrForbidden)
	}

	return u.rbacRepo.RemoveProjectMember(ctx, projectID, targetUserID)
}

