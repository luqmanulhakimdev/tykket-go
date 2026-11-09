package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/luqmanulhakimdev/tykket-go/internal/domain"
)

type mockProjectRepoForProjectUC struct {
	domain.ProjectRepository
	projects    map[string]*domain.Project
	projectsByID map[uuid.UUID]*domain.Project
}

func (m *mockProjectRepoForProjectUC) GetByKey(ctx context.Context, key string) (*domain.Project, error) {
	if p, ok := m.projects[key]; ok {
		return p, nil
	}
	return nil, domain.ErrNotFound
}

func (m *mockProjectRepoForProjectUC) GetByID(ctx context.Context, id uuid.UUID) (*domain.Project, error) {
	if p, ok := m.projectsByID[id]; ok {
		return p, nil
	}
	return nil, domain.ErrNotFound
}

func (m *mockProjectRepoForProjectUC) Create(ctx context.Context, p *domain.Project) error {
	m.projects[p.Key] = p
	m.projectsByID[p.ID] = p
	return nil
}

func (m *mockProjectRepoForProjectUC) ListByUserID(ctx context.Context, userID uuid.UUID) ([]domain.Project, error) {
	var list []domain.Project
	for _, p := range m.projectsByID {
		list = append(list, *p)
	}
	return list, nil
}

func (m *mockProjectRepoForProjectUC) Update(ctx context.Context, p *domain.Project) error {
	m.projectsByID[p.ID] = p
	return nil
}

func (m *mockProjectRepoForProjectUC) Delete(ctx context.Context, id uuid.UUID) error {
	delete(m.projectsByID, id)
	return nil
}

type mockRBACRepoForProjectUC struct {
	domain.RBACRepository
	members []domain.ProjectMember
}

func (m *mockRBACRepoForProjectUC) GetRoleByName(ctx context.Context, name string) (*domain.Role, error) {
	if name == "INVALID" {
		return nil, domain.ErrNotFound
	}
	return &domain.Role{
		ID:   uuid.New(),
		Name: name,
	}, nil
}

func (m *mockRBACRepoForProjectUC) AddProjectMember(ctx context.Context, member *domain.ProjectMember) error {
	m.members = append(m.members, *member)
	return nil
}

func (m *mockRBACRepoForProjectUC) ListProjectMembers(ctx context.Context, projectID uuid.UUID) ([]domain.ProjectMember, error) {
	return m.members, nil
}

func (m *mockRBACRepoForProjectUC) UpdateProjectMemberRole(ctx context.Context, projectID, userID, roleID uuid.UUID) error {
	return nil
}

func (m *mockRBACRepoForProjectUC) GetUserMemberInProject(ctx context.Context, userID, projectID uuid.UUID) (*domain.ProjectMember, error) {
	return &domain.ProjectMember{
		UserID:    userID,
		ProjectID: projectID,
	}, nil
}

func (m *mockRBACRepoForProjectUC) RemoveProjectMember(ctx context.Context, projectID, userID uuid.UUID) error {
	return nil
}

type mockUserRepoForProjectUC struct {
	domain.UserRepository
	users map[string]*domain.User
}

func (m *mockUserRepoForProjectUC) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	if u, ok := m.users[email]; ok {
		return u, nil
	}
	return nil, domain.ErrNotFound
}

func TestProjectUsecase_AllMethods(t *testing.T) {
	ownerID := uuid.New()
	otherUserID := uuid.New()
	existingProjectID := uuid.New()

	existingProject := &domain.Project{
		ID:      existingProjectID,
		Name:    "Existing Project",
		Key:     "EXIST",
		OwnerID: ownerID,
	}

	pRepo := &mockProjectRepoForProjectUC{
		projects: map[string]*domain.Project{
			"EXIST": existingProject,
		},
		projectsByID: map[uuid.UUID]*domain.Project{
			existingProjectID: existingProject,
		},
	}
	rbacRepo := &mockRBACRepoForProjectUC{}
	userRepo := &mockUserRepoForProjectUC{
		users: map[string]*domain.User{
			"colleague@example.com": {
				ID:    uuid.New(),
				Email: "colleague@example.com",
				Name:  "Colleague",
			},
		},
	}

	uc := NewProjectUsecase(pRepo, rbacRepo, userRepo)
	ctx := context.Background()

	t.Run("Create validation failure", func(t *testing.T) {
		_, err := uc.CreateProject(ctx, ownerID, domain.CreateProjectRequest{
			Name: "Bad Key",
			Key:  "toolongandlowercasename123",
		})
		if !errors.Is(err, domain.ErrInvalidInput) {
			t.Fatalf("expected ErrInvalidInput, got %v", err)
		}
	})

	t.Run("Create conflict key", func(t *testing.T) {
		_, err := uc.CreateProject(ctx, ownerID, domain.CreateProjectRequest{
			Name: "Duplicate Key",
			Key:  "EXIST",
		})
		if !errors.Is(err, domain.ErrConflict) {
			t.Fatalf("expected ErrConflict, got %v", err)
		}
	})

	t.Run("Create success", func(t *testing.T) {
		p, err := uc.CreateProject(ctx, ownerID, domain.CreateProjectRequest{
			Name: "New Valid Project",
			Key:  "NEWP",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if p.Key != "NEWP" {
			t.Errorf("expected key NEWP, got %s", p.Key)
		}
	})

	t.Run("GetProjectByID success and not found", func(t *testing.T) {
		p, err := uc.GetProjectByID(ctx, ownerID, existingProjectID)
		if err != nil || p.ID != existingProjectID {
			t.Fatalf("expected project retrieved, got %v", err)
		}

		_, err = uc.GetProjectByID(ctx, ownerID, uuid.New())
		if !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("expected ErrNotFound, got %v", err)
		}
	})

	t.Run("ListProjects success", func(t *testing.T) {
		list, err := uc.ListProjects(ctx, ownerID)
		if err != nil || len(list) == 0 {
			t.Fatalf("expected projects listed, got %v", err)
		}
	})

	t.Run("UpdateProject success and not found", func(t *testing.T) {
		updated, err := uc.UpdateProject(ctx, ownerID, existingProjectID, domain.UpdateProjectRequest{
			Name: "Updated Project Name",
		})
		if err != nil || updated.Name != "Updated Project Name" {
			t.Fatalf("expected updated project, got %v", err)
		}

		_, err = uc.UpdateProject(ctx, ownerID, uuid.New(), domain.UpdateProjectRequest{
			Name: "Random",
		})
		if !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("expected ErrNotFound, got %v", err)
		}
	})

	t.Run("DeleteProject forbidden if not owner", func(t *testing.T) {
		err := uc.DeleteProject(ctx, otherUserID, existingProjectID)
		if !errors.Is(err, domain.ErrForbidden) {
			t.Fatalf("expected ErrForbidden, got %v", err)
		}
	})

	t.Run("DeleteProject success", func(t *testing.T) {
		toDelete := &domain.Project{
			ID:      uuid.New(),
			Key:     "DEL",
			OwnerID: ownerID,
		}
		_ = pRepo.Create(ctx, toDelete)

		err := uc.DeleteProject(ctx, ownerID, toDelete.ID)
		if err != nil {
			t.Fatalf("unexpected error deleting project: %v", err)
		}
	})

	t.Run("Member management list, add, update, remove", func(t *testing.T) {
		// List
		_, err := uc.ListMembers(ctx, existingProjectID)
		if err != nil {
			t.Fatalf("unexpected error listing members: %v", err)
		}

		// Add member not found
		_, err = uc.AddMember(ctx, existingProjectID, domain.AddMemberRequest{
			Email:    "nonexistent@example.com",
			RoleName: "MEMBER",
		})
		if !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("expected user ErrNotFound, got %v", err)
		}

		// Add member invalid role
		_, err = uc.AddMember(ctx, existingProjectID, domain.AddMemberRequest{
			Email:    "colleague@example.com",
			RoleName: "INVALID",
		})
		if !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("expected role ErrNotFound, got %v", err)
		}

		// Add member success
		m, err := uc.AddMember(ctx, existingProjectID, domain.AddMemberRequest{
			Email:    "colleague@example.com",
			RoleName: "MEMBER",
		})
		if err != nil || m == nil {
			t.Fatalf("failed to add member: %v", err)
		}

		// Update member role invalid
		_, err = uc.UpdateMemberRole(ctx, existingProjectID, otherUserID, domain.UpdateMemberRoleRequest{
			RoleName: "INVALID",
		})
		if !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("expected role ErrNotFound, got %v", err)
		}

		// Update member role success
		_, err = uc.UpdateMemberRole(ctx, existingProjectID, otherUserID, domain.UpdateMemberRoleRequest{
			RoleName: "ADMIN",
		})
		if err != nil {
			t.Fatalf("unexpected error updating role: %v", err)
		}

		// Cannot remove owner
		err = uc.RemoveMember(ctx, existingProjectID, ownerID)
		if !errors.Is(err, domain.ErrForbidden) {
			t.Fatalf("expected ErrForbidden when removing owner, got %v", err)
		}

		// Remove other member success
		err = uc.RemoveMember(ctx, existingProjectID, otherUserID)
		if err != nil {
			t.Fatalf("unexpected error removing member: %v", err)
		}
	})
}
