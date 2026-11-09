package domain

import (
	"context"
	"io"

	"github.com/google/uuid"
)

type StatusDistribution struct {
	Status IssueStatus `json:"status"`
	Count  int64       `json:"count"`
	Points int64       `json:"points"`
}

type PriorityDistribution struct {
	Priority IssuePriority `json:"priority"`
	Count    int64         `json:"count"`
}

type SprintBurndownPoint struct {
	Date            string `json:"date"`
	RemainingPoints int    `json:"remaining_points"`
	IdealPoints     float64 `json:"ideal_points"`
}

type SprintMetrics struct {
	SprintID           uuid.UUID            `json:"sprint_id"`
	SprintName         string               `json:"sprint_name"`
	TotalIssues        int64                `json:"total_issues"`
	CompletedIssues    int64                `json:"completed_issues"`
	TotalPoints        int64                `json:"total_points"`
	CompletedPoints    int64                `json:"completed_points"`
	StatusDistribution []StatusDistribution `json:"status_distribution"`
	Burndown           []SprintBurndownPoint `json:"burndown"`
}

type ProjectSummaryMetrics struct {
	ProjectID            uuid.UUID              `json:"project_id"`
	TotalIssues          int64                  `json:"total_issues"`
	TotalSprints         int64                  `json:"total_sprints"`
	StatusDistribution   []StatusDistribution   `json:"status_distribution"`
	PriorityDistribution []PriorityDistribution `json:"priority_distribution"`
}

type MetricRepository interface {
	GetSprintMetrics(ctx context.Context, sprintID uuid.UUID) (*SprintMetrics, error)
	GetProjectMetrics(ctx context.Context, projectID uuid.UUID) (*ProjectSummaryMetrics, error)
}

type MetricUsecase interface {
	GetSprintMetrics(ctx context.Context, projectID, sprintID uuid.UUID) (*SprintMetrics, error)
	GetProjectMetrics(ctx context.Context, projectID uuid.UUID) (*ProjectSummaryMetrics, error)
	ExportIssuesCSV(ctx context.Context, filter IssueFilter, writer io.Writer) error
	ExportProjectSummary(ctx context.Context, projectID uuid.UUID) (interface{}, error)
}
