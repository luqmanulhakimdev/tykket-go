package usecase

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/luqmanulhakimdev/tykket-go/internal/domain"
)

type mockMetricRepo struct {
	domain.MetricRepository
}

func (m *mockMetricRepo) GetSprintMetrics(ctx context.Context, sprintID uuid.UUID) (*domain.SprintMetrics, error) {
	return &domain.SprintMetrics{
		SprintID:    sprintID,
		SprintName:  "Sprint Test",
		TotalIssues: 5,
	}, nil
}

func (m *mockMetricRepo) GetProjectMetrics(ctx context.Context, projectID uuid.UUID) (*domain.ProjectSummaryMetrics, error) {
	return &domain.ProjectSummaryMetrics{
		ProjectID:   projectID,
		TotalIssues: 10,
	}, nil
}

func TestMetricUsecase(t *testing.T) {
	projectID := uuid.New()
	sprintID := uuid.New()
	otherProjectID := uuid.New()

	metricRepo := &mockMetricRepo{}
	projectRepo := &mockProjectRepoForProjectUC{
		projects: map[string]*domain.Project{
			"TYK": {ID: projectID, Key: "TYK"},
		},
		projectsByID: map[uuid.UUID]*domain.Project{
			projectID: {ID: projectID, Key: "TYK", Name: "Tykket"},
		},
	}
	sprintRepo := &mockSprintRepoForSprintUC{
		sprints: map[uuid.UUID]*domain.Sprint{
			sprintID: {ID: sprintID, ProjectID: projectID, Name: "Sprint 1"},
		},
	}
	points := 5
	issueRepo := &mockIssueRepo{
		issues: map[uuid.UUID]*domain.Issue{
			uuid.New(): {
				Key:         "TYK-1",
				Type:        domain.IssueTypeTask,
				Status:      domain.IssueStatusTodo,
				Priority:    domain.IssuePriorityHigh,
				Title:       "Test Issue 1",
				StoryPoints: &points,
				Reporter:    &domain.User{Name: "Reporter Name"},
				Assignee:    &domain.User{Name: "Assignee Name"},
			},
		},
	}

	uc := NewMetricUsecase(metricRepo, issueRepo, projectRepo, sprintRepo)
	ctx := context.Background()

	t.Run("GetSprintMetrics success", func(t *testing.T) {
		m, err := uc.GetSprintMetrics(ctx, projectID, sprintID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if m.TotalIssues != 5 {
			t.Errorf("expected 5 issues, got %d", m.TotalIssues)
		}
	})

	t.Run("GetSprintMetrics wrong project", func(t *testing.T) {
		_, err := uc.GetSprintMetrics(ctx, otherProjectID, sprintID)
		if err == nil {
			t.Fatal("expected error for mismatch project ID, got nil")
		}
	})

	t.Run("GetProjectMetrics success and not found", func(t *testing.T) {
		m, err := uc.GetProjectMetrics(ctx, projectID)
		if err != nil || m.TotalIssues != 10 {
			t.Fatalf("unexpected error getting project metrics: %v", err)
		}

		_, err = uc.GetProjectMetrics(ctx, uuid.New())
		if err == nil {
			t.Fatal("expected error for missing project, got nil")
		}
	})

	t.Run("ExportIssuesCSV writes header and rows", func(t *testing.T) {
		var buf bytes.Buffer
		err := uc.ExportIssuesCSV(ctx, domain.IssueFilter{ProjectID: projectID}, &buf)
		if err != nil {
			t.Fatalf("unexpected error exporting CSV: %v", err)
		}

		out := buf.String()
		if !strings.Contains(out, "Key,Type,Status,Priority") || !strings.Contains(out, "Assignee Name") {
			t.Errorf("missing CSV headers or row content: %s", out)
		}
	})

	t.Run("ExportProjectSummary success and not found", func(t *testing.T) {
		summary, err := uc.ExportProjectSummary(ctx, projectID)
		if err != nil || summary == nil {
			t.Fatalf("failed to export summary: %v", err)
		}

		_, err = uc.ExportProjectSummary(ctx, uuid.New())
		if err == nil {
			t.Fatal("expected error exporting missing project, got nil")
		}
	})
}
