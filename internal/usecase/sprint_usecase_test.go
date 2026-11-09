package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/luqmanulhakimdev/tykket-go/internal/domain"
)

type mockSprintRepoForSprintUC struct {
	domain.SprintRepository
	sprints      map[uuid.UUID]*domain.Sprint
	activeSprint *domain.Sprint
}

func (m *mockSprintRepoForSprintUC) Create(ctx context.Context, sprint *domain.Sprint) error {
	m.sprints[sprint.ID] = sprint
	return nil
}

func (m *mockSprintRepoForSprintUC) GetByID(ctx context.Context, id uuid.UUID) (*domain.Sprint, error) {
	if s, ok := m.sprints[id]; ok {
		return s, nil
	}
	return nil, domain.ErrNotFound
}

func (m *mockSprintRepoForSprintUC) GetActiveByProjectID(ctx context.Context, projectID uuid.UUID) (*domain.Sprint, error) {
	return m.activeSprint, nil
}

func (m *mockSprintRepoForSprintUC) ListByProjectID(ctx context.Context, projectID uuid.UUID) ([]domain.Sprint, error) {
	var list []domain.Sprint
	for _, s := range m.sprints {
		if s.ProjectID == projectID {
			list = append(list, *s)
		}
	}
	return list, nil
}

func (m *mockSprintRepoForSprintUC) Update(ctx context.Context, sprint *domain.Sprint) error {
	m.sprints[sprint.ID] = sprint
	if sprint.Status == domain.SprintStatusActive {
		m.activeSprint = sprint
	} else if m.activeSprint != nil && m.activeSprint.ID == sprint.ID {
		m.activeSprint = nil
	}
	return nil
}

func (m *mockSprintRepoForSprintUC) Delete(ctx context.Context, id uuid.UUID) error {
	delete(m.sprints, id)
	return nil
}

type mockIssueRepoForSprintUC struct {
	domain.IssueRepository
}

func (m *mockIssueRepoForSprintUC) MoveIncompleteIssues(ctx context.Context, fromSprintID uuid.UUID, toSprintID *uuid.UUID) error {
	return nil
}

func (m *mockIssueRepoForSprintUC) BulkMoveIssues(ctx context.Context, issueIDs []uuid.UUID, sprintID *uuid.UUID) error {
	return nil
}

func TestSprintUsecase_AllMethods(t *testing.T) {
	projectID := uuid.New()
	otherProjectID := uuid.New()
	sprint1ID := uuid.New()
	sprint2ID := uuid.New()

	sprint1 := &domain.Sprint{
		ID:        sprint1ID,
		ProjectID: projectID,
		Name:      "Sprint 1",
		Status:    domain.SprintStatusPlanned,
	}

	sprint2 := &domain.Sprint{
		ID:        sprint2ID,
		ProjectID: projectID,
		Name:      "Sprint 2",
		Status:    domain.SprintStatusPlanned,
	}

	sprintRepo := &mockSprintRepoForSprintUC{
		sprints: map[uuid.UUID]*domain.Sprint{
			sprint1ID: sprint1,
			sprint2ID: sprint2,
		},
	}
	issueRepo := &mockIssueRepoForSprintUC{}

	uc := NewSprintUsecase(sprintRepo, issueRepo)
	ctx := context.Background()

	t.Run("CreateSprint success", func(t *testing.T) {
		s, err := uc.CreateSprint(ctx, projectID, domain.CreateSprintRequest{
			Name: "Sprint 3",
		})
		if err != nil || s.Name != "Sprint 3" {
			t.Fatalf("failed to create sprint: %v", err)
		}
	})

	t.Run("GetSprintByID success, not found, wrong project", func(t *testing.T) {
		s, err := uc.GetSprintByID(ctx, projectID, sprint1ID)
		if err != nil || s.ID != sprint1ID {
			t.Fatalf("expected sprint retrieved, got %v", err)
		}

		_, err = uc.GetSprintByID(ctx, otherProjectID, sprint1ID)
		if !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("expected ErrNotFound for wrong project, got %v", err)
		}

		_, err = uc.GetSprintByID(ctx, projectID, uuid.New())
		if !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("expected ErrNotFound for unknown ID, got %v", err)
		}
	})

	t.Run("ListSprints success", func(t *testing.T) {
		list, err := uc.ListSprints(ctx, projectID)
		if err != nil || len(list) == 0 {
			t.Fatalf("expected sprints list, got %v", err)
		}
	})

	t.Run("UpdateSprint success and not found", func(t *testing.T) {
		updated, err := uc.UpdateSprint(ctx, projectID, sprint1ID, domain.UpdateSprintRequest{
			Name: "Sprint 1 Renamed",
		})
		if err != nil || updated.Name != "Sprint 1 Renamed" {
			t.Fatalf("expected updated sprint, got %v", err)
		}

		_, err = uc.UpdateSprint(ctx, projectID, uuid.New(), domain.UpdateSprintRequest{
			Name: "Random",
		})
		if !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("expected ErrNotFound, got %v", err)
		}
	})

	t.Run("Start, Complete and Active Sprint Rules", func(t *testing.T) {
		// 1. Start Sprint 1 -> Success
		started1, err := uc.StartSprint(ctx, projectID, sprint1ID, domain.StartSprintRequest{})
		if err != nil || started1.Status != domain.SprintStatusActive {
			t.Fatalf("expected ACTIVE status, got %v", err)
		}

		// 2. Start Sprint 1 again -> returns same sprint
		startedAgain, err := uc.StartSprint(ctx, projectID, sprint1ID, domain.StartSprintRequest{})
		if err != nil || startedAgain.Status != domain.SprintStatusActive {
			t.Fatalf("expected same active sprint, got %v", err)
		}

		// 3. Start Sprint 2 while Sprint 1 is active -> fails with ErrActiveSprintExists
		_, err = uc.StartSprint(ctx, projectID, sprint2ID, domain.StartSprintRequest{})
		if !errors.Is(err, domain.ErrActiveSprintExists) {
			t.Fatalf("expected ErrActiveSprintExists, got %v", err)
		}

		// 4. Complete Sprint 2 (which is PLANNED, not active) -> fails
		_, err = uc.CompleteSprint(ctx, projectID, sprint2ID, domain.CompleteSprintRequest{})
		if !errors.Is(err, domain.ErrInvalidInput) {
			t.Fatalf("expected ErrInvalidInput, got %v", err)
		}

		// 5. Complete Sprint 1 -> Success
		completed1, err := uc.CompleteSprint(ctx, projectID, sprint1ID, domain.CompleteSprintRequest{})
		if err != nil || completed1.Status != domain.SprintStatusCompleted {
			t.Fatalf("expected COMPLETED status, got %v", err)
		}

		// 6. Start an already completed sprint -> fails
		_, err = uc.StartSprint(ctx, projectID, sprint1ID, domain.StartSprintRequest{})
		if !errors.Is(err, domain.ErrInvalidInput) {
			t.Fatalf("expected ErrInvalidInput when starting completed sprint, got %v", err)
		}

		// 7. Now Sprint 2 can be started
		started2, err := uc.StartSprint(ctx, projectID, sprint2ID, domain.StartSprintRequest{})
		if err != nil || started2.Status != domain.SprintStatusActive {
			t.Fatalf("expected ACTIVE status for sprint 2, got %v", err)
		}
	})

	t.Run("DeleteSprint success", func(t *testing.T) {
		toDeleteID := uuid.New()
		_ = sprintRepo.Create(ctx, &domain.Sprint{
			ID:        toDeleteID,
			ProjectID: projectID,
			Name:      "To Delete",
		})
		err := uc.DeleteSprint(ctx, projectID, toDeleteID)
		if err != nil {
			t.Fatalf("unexpected error deleting sprint: %v", err)
		}
	})

	t.Run("BulkMoveIssues tests", func(t *testing.T) {
		// Empty IDs returns nil immediately
		err := uc.BulkMoveIssues(ctx, projectID, domain.BulkMoveSprintIssuesRequest{
			IssueIDs: []uuid.UUID{},
		})
		if err != nil {
			t.Fatalf("expected nil for empty ids, got %v", err)
		}

		// Valid move
		err = uc.BulkMoveIssues(ctx, projectID, domain.BulkMoveSprintIssuesRequest{
			IssueIDs: []uuid.UUID{uuid.New()},
			SprintID: &sprint1ID,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Target sprint in different project
		otherSprintID := uuid.New()
		sprintRepo.sprints[otherSprintID] = &domain.Sprint{
			ID:        otherSprintID,
			ProjectID: otherProjectID,
		}
		err = uc.BulkMoveIssues(ctx, projectID, domain.BulkMoveSprintIssuesRequest{
			IssueIDs: []uuid.UUID{uuid.New()},
			SprintID: &otherSprintID,
		})
		if !errors.Is(err, domain.ErrForbidden) {
			t.Fatalf("expected ErrForbidden for foreign sprint, got %v", err)
		}
	})
}
