package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/luqmanulhakimdev/tykket-go/internal/domain"
)

type sprintUsecase struct {
	sprintRepo domain.SprintRepository
	issueRepo  domain.IssueRepository
}

func NewSprintUsecase(sprintRepo domain.SprintRepository, issueRepo domain.IssueRepository) domain.SprintUsecase {
	return &sprintUsecase{
		sprintRepo: sprintRepo,
		issueRepo:  issueRepo,
	}
}

func (u *sprintUsecase) CreateSprint(ctx context.Context, projectID uuid.UUID, req domain.CreateSprintRequest) (*domain.Sprint, error) {
	sprint := &domain.Sprint{
		ID:        uuid.New(),
		ProjectID: projectID,
		Name:      strings.TrimSpace(req.Name),
		Goal:      req.Goal,
		StartDate: req.StartDate,
		EndDate:   req.EndDate,
		Status:    domain.SprintStatusPlanned,
	}

	if err := u.sprintRepo.Create(ctx, sprint); err != nil {
		return nil, err
	}

	return sprint, nil
}

func (u *sprintUsecase) GetSprintByID(ctx context.Context, projectID, sprintID uuid.UUID) (*domain.Sprint, error) {
	sprint, err := u.sprintRepo.GetByID(ctx, sprintID)
	if err != nil {
		return nil, err
	}

	if sprint.ProjectID != projectID {
		return nil, domain.ErrNotFound
	}

	return sprint, nil
}

func (u *sprintUsecase) ListSprints(ctx context.Context, projectID uuid.UUID) ([]domain.Sprint, error) {
	return u.sprintRepo.ListByProjectID(ctx, projectID)
}

func (u *sprintUsecase) UpdateSprint(ctx context.Context, projectID, sprintID uuid.UUID, req domain.UpdateSprintRequest) (*domain.Sprint, error) {
	sprint, err := u.GetSprintByID(ctx, projectID, sprintID)
	if err != nil {
		return nil, err
	}

	sprint.Name = strings.TrimSpace(req.Name)
	sprint.Goal = req.Goal
	sprint.StartDate = req.StartDate
	sprint.EndDate = req.EndDate

	if err := u.sprintRepo.Update(ctx, sprint); err != nil {
		return nil, err
	}

	return sprint, nil
}

func (u *sprintUsecase) StartSprint(ctx context.Context, projectID, sprintID uuid.UUID, req domain.StartSprintRequest) (*domain.Sprint, error) {
	sprint, err := u.GetSprintByID(ctx, projectID, sprintID)
	if err != nil {
		return nil, err
	}

	if sprint.Status == domain.SprintStatusActive {
		return sprint, nil
	}
	if sprint.Status == domain.SprintStatusCompleted {
		return nil, fmt.Errorf("%w: cannot start an already completed sprint", domain.ErrInvalidInput)
	}

	// Business rule check: only 1 active sprint allowed per project
	activeSprint, err := u.sprintRepo.GetActiveByProjectID(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if activeSprint != nil && activeSprint.ID != sprintID {
		return nil, domain.ErrActiveSprintExists
	}

	now := time.Now().UTC()
	if req.StartDate != nil {
		sprint.StartDate = req.StartDate
	} else if sprint.StartDate == nil {
		sprint.StartDate = &now
	}

	if req.EndDate != nil {
		sprint.EndDate = req.EndDate
	}

	sprint.Status = domain.SprintStatusActive
	if err := u.sprintRepo.Update(ctx, sprint); err != nil {
		return nil, err
	}

	return sprint, nil
}

func (u *sprintUsecase) CompleteSprint(ctx context.Context, projectID, sprintID uuid.UUID, req domain.CompleteSprintRequest) (*domain.Sprint, error) {
	sprint, err := u.GetSprintByID(ctx, projectID, sprintID)
	if err != nil {
		return nil, err
	}

	if sprint.Status != domain.SprintStatusActive {
		return nil, fmt.Errorf("%w: only active sprints can be completed", domain.ErrInvalidInput)
	}

	// Reconcile incomplete issues: move to target sprint or Backlog (nil)
	if err := u.issueRepo.MoveIncompleteIssues(ctx, sprintID, req.MoveIncompleteToSprintID); err != nil {
		return nil, fmt.Errorf("failed to reconcile incomplete issues: %w", err)
	}

	now := time.Now().UTC()
	sprint.EndDate = &now
	sprint.Status = domain.SprintStatusCompleted

	if err := u.sprintRepo.Update(ctx, sprint); err != nil {
		return nil, err
	}

	return sprint, nil
}

func (u *sprintUsecase) DeleteSprint(ctx context.Context, projectID, sprintID uuid.UUID) error {
	sprint, err := u.GetSprintByID(ctx, projectID, sprintID)
	if err != nil {
		return err
	}

	// Move all issues belonging to this sprint back to Backlog
	var nilSprint *uuid.UUID = nil
	_ = u.issueRepo.MoveIncompleteIssues(ctx, sprint.ID, nilSprint)

	return u.sprintRepo.Delete(ctx, sprintID)
}

func (u *sprintUsecase) BulkMoveIssues(ctx context.Context, projectID uuid.UUID, req domain.BulkMoveSprintIssuesRequest) error {
	if len(req.IssueIDs) == 0 {
		return nil
	}

	if req.SprintID != nil {
		targetSprint, err := u.sprintRepo.GetByID(ctx, *req.SprintID)
		if err != nil {
			return err
		}
		if targetSprint.ProjectID != projectID {
			return domain.ErrForbidden
		}
	}

	return u.issueRepo.BulkMoveIssues(ctx, req.IssueIDs, req.SprintID)
}

