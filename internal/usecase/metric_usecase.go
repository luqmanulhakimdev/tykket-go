package usecase

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/luqmanulhakimdev/tykket-go/internal/domain"
)

type metricUsecase struct {
	metricRepo  domain.MetricRepository
	issueRepo   domain.IssueRepository
	projectRepo domain.ProjectRepository
	sprintRepo  domain.SprintRepository
}

func NewMetricUsecase(
	metricRepo domain.MetricRepository,
	issueRepo domain.IssueRepository,
	projectRepo domain.ProjectRepository,
	sprintRepo domain.SprintRepository,
) domain.MetricUsecase {
	return &metricUsecase{
		metricRepo:  metricRepo,
		issueRepo:   issueRepo,
		projectRepo: projectRepo,
		sprintRepo:  sprintRepo,
	}
}

func (u *metricUsecase) GetSprintMetrics(ctx context.Context, projectID, sprintID uuid.UUID) (*domain.SprintMetrics, error) {
	sprint, err := u.sprintRepo.GetByID(ctx, sprintID)
	if err != nil {
		return nil, err
	}
	if sprint.ProjectID != projectID {
		return nil, domain.ErrNotFound
	}

	return u.metricRepo.GetSprintMetrics(ctx, sprintID)
}

func (u *metricUsecase) GetProjectMetrics(ctx context.Context, projectID uuid.UUID) (*domain.ProjectSummaryMetrics, error) {
	_, err := u.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		return nil, err
	}

	return u.metricRepo.GetProjectMetrics(ctx, projectID)
}

func (u *metricUsecase) ExportIssuesCSV(ctx context.Context, filter domain.IssueFilter, writer io.Writer) error {
	csvWriter := csv.NewWriter(writer)
	defer csvWriter.Flush()

	// Write CSV Header
	headers := []string{
		"Key", "Type", "Status", "Priority", "Title",
		"Story Points", "Assignee", "Reporter", "Created At",
	}
	if err := csvWriter.Write(headers); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Fetch without pagination limit for export
	filter.Limit = 10000
	filter.Offset = 0
	issues, _, err := u.issueRepo.List(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to list issues for export: %w", err)
	}

	for _, issue := range issues {
		assigneeName := "Unassigned"
		if issue.Assignee != nil {
			assigneeName = issue.Assignee.Name
		}
		reporterName := ""
		if issue.Reporter != nil {
			reporterName = issue.Reporter.Name
		}
		pointsStr := ""
		if issue.StoryPoints != nil {
			pointsStr = strconv.Itoa(*issue.StoryPoints)
		}

		row := []string{
			issue.Key,
			string(issue.Type),
			string(issue.Status),
			string(issue.Priority),
			issue.Title,
			pointsStr,
			assigneeName,
			reporterName,
			issue.CreatedAt.Format(time.RFC3339),
		}

		if err := csvWriter.Write(row); err != nil {
			return fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	return nil
}

func (u *metricUsecase) ExportProjectSummary(ctx context.Context, projectID uuid.UUID) (interface{}, error) {
	project, err := u.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		return nil, err
	}

	metrics, err := u.metricRepo.GetProjectMetrics(ctx, projectID)
	if err != nil {
		return nil, err
	}

	sprints, _ := u.sprintRepo.ListByProjectID(ctx, projectID)

	return map[string]interface{}{
		"project":   project,
		"metrics":   metrics,
		"sprints":   sprints,
		"timestamp": time.Now().UTC(),
	}, nil
}

