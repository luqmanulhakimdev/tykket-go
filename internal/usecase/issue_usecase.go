package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/luqmanulhakimdev/tykket-go/internal/domain"
)

type issueUsecase struct {
	issueRepo   domain.IssueRepository
	projectRepo domain.ProjectRepository
	sprintRepo  domain.SprintRepository
}

func NewIssueUsecase(
	issueRepo domain.IssueRepository,
	projectRepo domain.ProjectRepository,
	sprintRepo domain.SprintRepository,
) domain.IssueUsecase {
	return &issueUsecase{
		issueRepo:   issueRepo,
		projectRepo: projectRepo,
		sprintRepo:  sprintRepo,
	}
}

func (u *issueUsecase) validateHierarchy(ctx context.Context, projectID uuid.UUID, issueType domain.IssueType, parentID *uuid.UUID) error {
	switch issueType {
	case domain.IssueTypeEpic:
		if parentID != nil {
			return fmt.Errorf("%w: EPIC cannot have a parent issue", domain.ErrInvalidHierarchy)
		}

	case domain.IssueTypeStory, domain.IssueTypeTask, domain.IssueTypeBug:
		if parentID != nil {
			parent, err := u.issueRepo.GetByID(ctx, *parentID)
			if err != nil {
				return fmt.Errorf("%w: parent issue not found", domain.ErrNotFound)
			}
			if parent.ProjectID != projectID {
				return fmt.Errorf("%w: parent issue belongs to a different project", domain.ErrInvalidHierarchy)
			}
			if parent.Type != domain.IssueTypeEpic {
				return fmt.Errorf("%w: %s parent must be an EPIC (found %s)", domain.ErrInvalidHierarchy, issueType, parent.Type)
			}
		}

	case domain.IssueTypeSubtask:
		if parentID == nil {
			return fmt.Errorf("%w: SUBTASK must have a parent issue (STORY or TASK)", domain.ErrInvalidHierarchy)
		}
		parent, err := u.issueRepo.GetByID(ctx, *parentID)
		if err != nil {
			return fmt.Errorf("%w: parent issue not found", domain.ErrNotFound)
		}
		if parent.ProjectID != projectID {
			return fmt.Errorf("%w: parent issue belongs to a different project", domain.ErrInvalidHierarchy)
		}
		if parent.Type != domain.IssueTypeStory && parent.Type != domain.IssueTypeTask {
			return fmt.Errorf("%w: SUBTASK parent must be a STORY or TASK (found %s)", domain.ErrInvalidHierarchy, parent.Type)
		}
	}

	return nil
}

func (u *issueUsecase) CreateIssue(ctx context.Context, reporterID, projectID uuid.UUID, req domain.CreateIssueRequest) (*domain.Issue, error) {
	// 1. Hierarchy validation
	if err := u.validateHierarchy(ctx, projectID, req.Type, req.ParentID); err != nil {
		return nil, err
	}

	// 2. Sprint validation (if assigned)
	if req.SprintID != nil {
		sprint, err := u.sprintRepo.GetByID(ctx, *req.SprintID)
		if err != nil {
			return nil, fmt.Errorf("%w: sprint not found", domain.ErrNotFound)
		}
		if sprint.ProjectID != projectID {
			return nil, fmt.Errorf("%w: sprint does not belong to project", domain.ErrForbidden)
		}
	}

	// 3. Atomic issue counter increment
	prefixKey, counter, err := u.projectRepo.IncrementIssueCounter(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate issue number: %w", err)
	}

	key := fmt.Sprintf("%s-%d", prefixKey, counter)

	position := 65535.0
	if req.Position != nil && *req.Position > 0 {
		position = *req.Position
	}

	status := domain.IssueStatusTodo
	if req.Status != nil {
		status = *req.Status
	}

	issue := &domain.Issue{
		ID:          uuid.New(),
		ProjectID:   projectID,
		SprintID:    req.SprintID,
		ParentID:    req.ParentID,
		Key:         key,
		IssueNumber: counter,
		Type:        req.Type,
		Status:      status,
		Priority:    req.Priority,
		Title:       strings.TrimSpace(req.Title),
		Description: req.Description,
		ReporterID:  reporterID,
		AssigneeID:  req.AssigneeID,
		StoryPoints: req.StoryPoints,
		Position:    position,
	}

	if err := u.issueRepo.Create(ctx, issue); err != nil {
		return nil, err
	}

	return issue, nil
}

func (u *issueUsecase) GetIssueByID(ctx context.Context, projectID, issueID uuid.UUID) (*domain.Issue, error) {
	issue, err := u.issueRepo.GetByID(ctx, issueID)
	if err != nil {
		return nil, err
	}

	if issue.ProjectID != projectID {
		return nil, domain.ErrNotFound
	}

	return issue, nil
}

func (u *issueUsecase) GetIssueByKey(ctx context.Context, key string) (*domain.Issue, error) {
	return u.issueRepo.GetByKey(ctx, key)
}

func (u *issueUsecase) ListIssues(ctx context.Context, filter domain.IssueFilter) ([]domain.Issue, int64, error) {
	return u.issueRepo.List(ctx, filter)
}

func (u *issueUsecase) UpdateIssue(ctx context.Context, projectID, issueID uuid.UUID, req domain.UpdateIssueRequest) (*domain.Issue, error) {
	issue, err := u.GetIssueByID(ctx, projectID, issueID)
	if err != nil {
		return nil, err
	}

	targetType := issue.Type
	if req.Type != nil {
		targetType = *req.Type
	}

	targetParentID := issue.ParentID
	if req.ParentID != nil {
		targetParentID = req.ParentID
	}

	// Revalidate hierarchy if type or parent changed
	if req.Type != nil || req.ParentID != nil {
		if err := u.validateHierarchy(ctx, projectID, targetType, targetParentID); err != nil {
			return nil, err
		}
		issue.Type = targetType
		issue.ParentID = targetParentID
	}

	if req.Title != nil {
		issue.Title = strings.TrimSpace(*req.Title)
	}
	if req.Description != nil {
		issue.Description = req.Description
	}
	if req.Priority != nil {
		issue.Priority = *req.Priority
	}
	if req.AssigneeID != nil {
		issue.AssigneeID = req.AssigneeID
	}
	if req.StoryPoints != nil {
		issue.StoryPoints = req.StoryPoints
	}
	if req.SprintID != nil {
		if *req.SprintID == uuid.Nil {
			issue.SprintID = nil
		} else {
			sprint, err := u.sprintRepo.GetByID(ctx, *req.SprintID)
			if err != nil {
				return nil, fmt.Errorf("%w: sprint not found", domain.ErrNotFound)
			}
			if sprint.ProjectID != projectID {
				return nil, fmt.Errorf("%w: sprint does not belong to project", domain.ErrForbidden)
			}
			issue.SprintID = req.SprintID
		}
	}

	if err := u.issueRepo.Update(ctx, issue); err != nil {
		return nil, err
	}

	return issue, nil
}

func (u *issueUsecase) MoveIssue(ctx context.Context, projectID, issueID uuid.UUID, req domain.MoveIssueRequest) (*domain.Issue, error) {
	issue, err := u.GetIssueByID(ctx, projectID, issueID)
	if err != nil {
		return nil, err
	}

	if err := u.issueRepo.UpdatePositionAndStatus(ctx, issueID, req.Status, req.Position, req.SprintID); err != nil {
		return nil, err
	}

	if req.Status != nil {
		issue.Status = *req.Status
	}
	issue.Position = req.Position
	if req.SprintID != nil {
		issue.SprintID = req.SprintID
	}

	return issue, nil
}

func (u *issueUsecase) DeleteIssue(ctx context.Context, projectID, issueID uuid.UUID) error {
	_, err := u.GetIssueByID(ctx, projectID, issueID)
	if err != nil {
		return err
	}

	return u.issueRepo.Delete(ctx, issueID)
}

