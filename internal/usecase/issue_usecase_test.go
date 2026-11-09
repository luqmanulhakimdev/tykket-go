package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/luqmanulhakimdev/tykket-go/internal/domain"
)

type mockIssueRepo struct {
	domain.IssueRepository
	issues map[uuid.UUID]*domain.Issue
}

func (m *mockIssueRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Issue, error) {
	issue, exists := m.issues[id]
	if !exists {
		return nil, domain.ErrNotFound
	}
	return issue, nil
}

func (m *mockIssueRepo) GetByKey(ctx context.Context, key string) (*domain.Issue, error) {
	for _, issue := range m.issues {
		if issue.Key == key {
			return issue, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (m *mockIssueRepo) Create(ctx context.Context, issue *domain.Issue) error {
	m.issues[issue.ID] = issue
	return nil
}

func (m *mockIssueRepo) List(ctx context.Context, filter domain.IssueFilter) ([]domain.Issue, int64, error) {
	var list []domain.Issue
	for _, issue := range m.issues {
		list = append(list, *issue)
	}
	return list, int64(len(list)), nil
}

func (m *mockIssueRepo) Update(ctx context.Context, issue *domain.Issue) error {
	m.issues[issue.ID] = issue
	return nil
}

func (m *mockIssueRepo) UpdatePositionAndStatus(ctx context.Context, id uuid.UUID, status *domain.IssueStatus, position float64, sprintID *uuid.UUID) error {
	if issue, exists := m.issues[id]; exists {
		if status != nil {
			issue.Status = *status
		}
		issue.Position = position
		if sprintID != nil {
			issue.SprintID = sprintID
		}
		return nil
	}
	return domain.ErrNotFound
}

func (m *mockIssueRepo) Delete(ctx context.Context, id uuid.UUID) error {
	delete(m.issues, id)
	return nil
}

type mockProjectRepo struct {
	domain.ProjectRepository
	counter int64
}

func (m *mockProjectRepo) IncrementIssueCounter(ctx context.Context, projectID uuid.UUID) (string, int64, error) {
	m.counter++
	return "TYK", m.counter, nil
}

type mockSprintRepo struct {
	domain.SprintRepository
	sprints map[uuid.UUID]*domain.Sprint
}

func (m *mockSprintRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Sprint, error) {
	if s, ok := m.sprints[id]; ok {
		return s, nil
	}
	return nil, domain.ErrNotFound
}

func TestIssueUsecase_AllMethods(t *testing.T) {
	projectID := uuid.New()
	otherProjectID := uuid.New()
	reporterID := uuid.New()

	epicID := uuid.New()
	storyID := uuid.New()
	taskID := uuid.New()
	foreignEpicID := uuid.New()
	validSprintID := uuid.New()
	foreignSprintID := uuid.New()

	issueRepo := &mockIssueRepo{
		issues: map[uuid.UUID]*domain.Issue{
			epicID: {
				ID:        epicID,
				ProjectID: projectID,
				Key:       "TYK-1",
				Type:      domain.IssueTypeEpic,
			},
			storyID: {
				ID:        storyID,
				ProjectID: projectID,
				Key:       "TYK-2",
				Type:      domain.IssueTypeStory,
			},
			taskID: {
				ID:        taskID,
				ProjectID: projectID,
				Key:       "TYK-3",
				Type:      domain.IssueTypeTask,
			},
			foreignEpicID: {
				ID:        foreignEpicID,
				ProjectID: otherProjectID,
				Key:       "OTH-1",
				Type:      domain.IssueTypeEpic,
			},
		},
	}
	projectRepo := &mockProjectRepo{}
	sprintRepo := &mockSprintRepo{
		sprints: map[uuid.UUID]*domain.Sprint{
			validSprintID:   {ID: validSprintID, ProjectID: projectID},
			foreignSprintID: {ID: foreignSprintID, ProjectID: otherProjectID},
		},
	}

	uc := NewIssueUsecase(issueRepo, projectRepo, sprintRepo)
	ctx := context.Background()

	t.Run("Hierarchy and Sprint Validation", func(t *testing.T) {
		// EPIC cannot have parent
		_, err := uc.CreateIssue(ctx, reporterID, projectID, domain.CreateIssueRequest{
			Type:     domain.IssueTypeEpic,
			ParentID: &storyID,
			Title:    "Invalid Epic",
			Priority: domain.IssuePriorityHigh,
		})
		if !errors.Is(err, domain.ErrInvalidHierarchy) {
			t.Fatalf("expected ErrInvalidHierarchy, got %v", err)
		}

		// STORY parent not found
		unknownParent := uuid.New()
		_, err = uc.CreateIssue(ctx, reporterID, projectID, domain.CreateIssueRequest{
			Type:     domain.IssueTypeStory,
			ParentID: &unknownParent,
			Title:    "Story with unknown parent",
			Priority: domain.IssuePriorityLow,
		})
		if !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("expected ErrNotFound for missing parent, got %v", err)
		}

		// STORY parent in other project
		_, err = uc.CreateIssue(ctx, reporterID, projectID, domain.CreateIssueRequest{
			Type:     domain.IssueTypeStory,
			ParentID: &foreignEpicID,
			Title:    "Story with foreign parent",
			Priority: domain.IssuePriorityLow,
		})
		if !errors.Is(err, domain.ErrInvalidHierarchy) {
			t.Fatalf("expected ErrInvalidHierarchy for foreign parent, got %v", err)
		}

		// STORY parent must be EPIC (give it STORY as parent -> error)
		_, err = uc.CreateIssue(ctx, reporterID, projectID, domain.CreateIssueRequest{
			Type:     domain.IssueTypeStory,
			ParentID: &taskID,
			Title:    "Story with task parent",
			Priority: domain.IssuePriorityLow,
		})
		if !errors.Is(err, domain.ErrInvalidHierarchy) {
			t.Fatalf("expected ErrInvalidHierarchy for non-epic parent, got %v", err)
		}

		// SUBTASK must have parent
		_, err = uc.CreateIssue(ctx, reporterID, projectID, domain.CreateIssueRequest{
			Type:     domain.IssueTypeSubtask,
			ParentID: nil,
			Title:    "Invalid Subtask without parent",
			Priority: domain.IssuePriorityMedium,
		})
		if !errors.Is(err, domain.ErrInvalidHierarchy) {
			t.Fatalf("expected ErrInvalidHierarchy, got %v", err)
		}

		// SUBTASK parent not found
		_, err = uc.CreateIssue(ctx, reporterID, projectID, domain.CreateIssueRequest{
			Type:     domain.IssueTypeSubtask,
			ParentID: &unknownParent,
			Title:    "Subtask with unknown parent",
			Priority: domain.IssuePriorityMedium,
		})
		if !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("expected ErrNotFound for missing subtask parent, got %v", err)
		}

		// SUBTASK parent in other project
		_, err = uc.CreateIssue(ctx, reporterID, projectID, domain.CreateIssueRequest{
			Type:     domain.IssueTypeSubtask,
			ParentID: &foreignEpicID,
			Title:    "Subtask with foreign parent",
			Priority: domain.IssuePriorityMedium,
		})
		if !errors.Is(err, domain.ErrInvalidHierarchy) {
			t.Fatalf("expected ErrInvalidHierarchy for foreign subtask parent, got %v", err)
		}

		// SUBTASK cannot have EPIC as parent
		_, err = uc.CreateIssue(ctx, reporterID, projectID, domain.CreateIssueRequest{
			Type:     domain.IssueTypeSubtask,
			ParentID: &epicID,
			Title:    "Subtask under epic",
			Priority: domain.IssuePriorityMedium,
		})
		if !errors.Is(err, domain.ErrInvalidHierarchy) {
			t.Fatalf("expected ErrInvalidHierarchy, got %v", err)
		}

		// Sprint in different project fails
		_, err = uc.CreateIssue(ctx, reporterID, projectID, domain.CreateIssueRequest{
			Type:     domain.IssueTypeTask,
			SprintID: &foreignSprintID,
			Title:    "Task with foreign sprint",
			Priority: domain.IssuePriorityLow,
		})
		if !errors.Is(err, domain.ErrForbidden) {
			t.Fatalf("expected ErrForbidden for foreign sprint, got %v", err)
		}

		// Valid SUBTASK under TASK creates with atomic key
		subtask, err := uc.CreateIssue(ctx, reporterID, projectID, domain.CreateIssueRequest{
			Type:     domain.IssueTypeSubtask,
			ParentID: &taskID,
			Title:    "Valid Subtask under Task",
			Priority: domain.IssuePriorityMedium,
			SprintID: &validSprintID,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if subtask.Key == "" {
			t.Errorf("expected non-empty key")
		}
	})

	t.Run("GetIssueByID success, not found, wrong project", func(t *testing.T) {
		issue, err := uc.GetIssueByID(ctx, projectID, storyID)
		if err != nil || issue.ID != storyID {
			t.Fatalf("expected story retrieved, got %v", err)
		}

		_, err = uc.GetIssueByID(ctx, otherProjectID, storyID)
		if !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("expected ErrNotFound for other project, got %v", err)
		}

		_, err = uc.GetIssueByID(ctx, projectID, uuid.New())
		if !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("expected ErrNotFound for unknown ID, got %v", err)
		}
	})

	t.Run("GetIssueByKey success", func(t *testing.T) {
		issue, err := uc.GetIssueByKey(ctx, "TYK-2")
		if err != nil || issue.Key != "TYK-2" {
			t.Fatalf("failed to get issue by key: %v", err)
		}
	})

	t.Run("ListIssues success", func(t *testing.T) {
		list, total, err := uc.ListIssues(ctx, domain.IssueFilter{ProjectID: projectID})
		if err != nil || total == 0 || len(list) == 0 {
			t.Fatalf("expected list of issues, got %v", err)
		}
	})

	t.Run("UpdateIssue hierarchy error and full updates", func(t *testing.T) {
		// Update to SUBTASK without parent fails
		subtaskType := domain.IssueTypeSubtask
		var nilParent *uuid.UUID = nil
		_, err := uc.UpdateIssue(ctx, projectID, storyID, domain.UpdateIssueRequest{
			Type:     &subtaskType,
			ParentID: nilParent,
		})
		if !errors.Is(err, domain.ErrInvalidHierarchy) {
			t.Fatalf("expected ErrInvalidHierarchy updating to subtask without parent, got %v", err)
		}

		newTitle := "Updated Story Title"
		newDesc := "New description"
		newPriority := domain.IssuePriorityUrgent
		assignee := uuid.New()
		points := 8

		updated, err := uc.UpdateIssue(ctx, projectID, storyID, domain.UpdateIssueRequest{
			Title:       &newTitle,
			Description: &newDesc,
			Priority:    &newPriority,
			AssigneeID:  &assignee,
			StoryPoints: &points,
			SprintID:    &validSprintID,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if updated.Title != newTitle || updated.Priority != newPriority {
			t.Errorf("unexpected updated values: %+v", updated)
		}
	})

	t.Run("MoveIssue success with status, position and sprint", func(t *testing.T) {
		newStatus := domain.IssueStatusInProgress
		moved, err := uc.MoveIssue(ctx, projectID, storyID, domain.MoveIssueRequest{
			Status:   &newStatus,
			Position: 12345.67,
			SprintID: &validSprintID,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if moved.Status != newStatus || moved.Position != 12345.67 {
			t.Errorf("unexpected moved state: %+v", moved)
		}
	})

	t.Run("DeleteIssue success and not found", func(t *testing.T) {
		toDeleteID := uuid.New()
		issueRepo.issues[toDeleteID] = &domain.Issue{
			ID:        toDeleteID,
			ProjectID: projectID,
		}

		err := uc.DeleteIssue(ctx, projectID, toDeleteID)
		if err != nil {
			t.Fatalf("unexpected error deleting issue: %v", err)
		}

		err = uc.DeleteIssue(ctx, projectID, uuid.New())
		if !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("expected ErrNotFound for missing issue, got %v", err)
		}
	})
}
