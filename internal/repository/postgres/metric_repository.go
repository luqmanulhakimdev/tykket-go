package postgres

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/luqmanulhakimdev/tykket-go/internal/domain"
)

type metricRepository struct {
	pool *pgxpool.Pool
}

func NewMetricRepository(pool *pgxpool.Pool) domain.MetricRepository {
	return &metricRepository{pool: pool}
}

func (r *metricRepository) GetSprintMetrics(ctx context.Context, sprintID uuid.UUID) (*domain.SprintMetrics, error) {
	// 1. Fetch Sprint Details
	var (
		sprintName string
		startDate  *time.Time
		endDate    *time.Time
	)
	err := r.pool.QueryRow(ctx, "SELECT name, start_date, end_date FROM sprints WHERE id = $1 AND deleted_at IS NULL", sprintID).
		Scan(&sprintName, &startDate, &endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch sprint details for metrics: %w", err)
	}

	metrics := &domain.SprintMetrics{
		SprintID:           sprintID,
		SprintName:         sprintName,
		StatusDistribution: make([]domain.StatusDistribution, 0),
		Burndown:           make([]domain.SprintBurndownPoint, 0),
	}

	// 2. Fetch Status Aggregates
	query := `
		SELECT 
			status,
			COUNT(*) as count,
			COALESCE(SUM(story_points), 0) as points
		FROM issues
		WHERE sprint_id = $1 AND deleted_at IS NULL
		GROUP BY status
	`
	rows, err := r.pool.Query(ctx, query, sprintID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch sprint status distribution: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var dist domain.StatusDistribution
		if err := rows.Scan(&dist.Status, &dist.Count, &dist.Points); err != nil {
			return nil, fmt.Errorf("failed to scan status distribution: %w", err)
		}
		metrics.StatusDistribution = append(metrics.StatusDistribution, dist)
		metrics.TotalIssues += dist.Count
		metrics.TotalPoints += dist.Points

		if dist.Status == domain.IssueStatusDone {
			metrics.CompletedIssues = dist.Count
			metrics.CompletedPoints = dist.Points
		}
	}

	// 3. Compute Burndown line if start and end dates are set
	if startDate != nil && endDate != nil && endDate.After(*startDate) {
		totalDays := int(math.Ceil(endDate.Sub(*startDate).Hours() / 24))
		if totalDays < 1 {
			totalDays = 1
		}

		idealPerDay := float64(metrics.TotalPoints) / float64(totalDays)
		remainingPoints := int(metrics.TotalPoints - metrics.CompletedPoints)

		for i := 0; i <= totalDays; i++ {
			pointDate := startDate.Add(time.Duration(i*24) * time.Hour)
			ideal := math.Max(0, float64(metrics.TotalPoints)-(float64(i)*idealPerDay))

			metrics.Burndown = append(metrics.Burndown, domain.SprintBurndownPoint{
				Date:            pointDate.Format("2006-01-02"),
				RemainingPoints: remainingPoints,
				IdealPoints:     math.Round(ideal*100) / 100,
			})
		}
	}

	return metrics, nil
}

func (r *metricRepository) GetProjectMetrics(ctx context.Context, projectID uuid.UUID) (*domain.ProjectSummaryMetrics, error) {
	metrics := &domain.ProjectSummaryMetrics{
		ProjectID:            projectID,
		StatusDistribution:   make([]domain.StatusDistribution, 0),
		PriorityDistribution: make([]domain.PriorityDistribution, 0),
	}

	// 1. Total Sprints count
	_ = r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM sprints WHERE project_id = $1 AND deleted_at IS NULL", projectID).
		Scan(&metrics.TotalSprints)

	// 2. Status Distribution & Total Issues
	statusQuery := `
		SELECT 
			status,
			COUNT(*) as count,
			COALESCE(SUM(story_points), 0) as points
		FROM issues
		WHERE project_id = $1 AND deleted_at IS NULL
		GROUP BY status
	`
	rows, err := r.pool.Query(ctx, statusQuery, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to query project status metrics: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var dist domain.StatusDistribution
		if err := rows.Scan(&dist.Status, &dist.Count, &dist.Points); err != nil {
			return nil, fmt.Errorf("failed to scan project status distribution: %w", err)
		}
		metrics.StatusDistribution = append(metrics.StatusDistribution, dist)
		metrics.TotalIssues += dist.Count
	}

	// 3. Priority Distribution
	priorityQuery := `
		SELECT 
			priority,
			COUNT(*) as count
		FROM issues
		WHERE project_id = $1 AND deleted_at IS NULL
		GROUP BY priority
	`
	pRows, err := r.pool.Query(ctx, priorityQuery, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to query project priority metrics: %w", err)
	}
	defer pRows.Close()

	for pRows.Next() {
		var pDist domain.PriorityDistribution
		if err := pRows.Scan(&pDist.Priority, &pDist.Count); err != nil {
			return nil, fmt.Errorf("failed to scan priority distribution: %w", err)
		}
		metrics.PriorityDistribution = append(metrics.PriorityDistribution, pDist)
	}

	return metrics, nil
}

