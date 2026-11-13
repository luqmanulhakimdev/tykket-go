package usecase

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/luqmanulhakimdev/tykket-go/internal/domain"
)

type commentUsecase struct {
	commentRepo domain.CommentRepository
	issueRepo   domain.IssueRepository
	projectRepo domain.ProjectRepository
	rbacRepo    domain.RBACRepository
}

func NewCommentUsecase(
	commentRepo domain.CommentRepository,
	issueRepo domain.IssueRepository,
	projectRepo domain.ProjectRepository,
	rbacRepo domain.RBACRepository,
) domain.CommentUsecase {
	return &commentUsecase{
		commentRepo: commentRepo,
		issueRepo:   issueRepo,
		projectRepo: projectRepo,
		rbacRepo:    rbacRepo,
	}
}

func (u *commentUsecase) CreateComment(
	ctx context.Context,
	userID, issueID uuid.UUID,
	req domain.CreateCommentRequest,
) (*domain.Comment, error) {
	content := strings.TrimSpace(req.Content)
	if content == "" {
		return nil, domain.ErrInvalidInput
	}

	// Verify issue exists
	issue, err := u.issueRepo.GetByID(ctx, issueID)
	if err != nil {
		return nil, err
	}

	// If replying to another comment, verify parent comment exists
	if req.ParentID != nil {
		parentComment, err := u.commentRepo.GetByID(ctx, *req.ParentID)
		if err != nil {
			return nil, err
		}
		if parentComment.IssueID != issue.ID {
			return nil, domain.ErrInvalidInput
		}
	}

	comment := &domain.Comment{
		IssueID:  issue.ID,
		UserID:   userID,
		ParentID: req.ParentID,
		Content:  content,
	}

	if err := u.commentRepo.Create(ctx, comment); err != nil {
		return nil, err
	}

	// Return comment with populated author details
	return u.commentRepo.GetByID(ctx, comment.ID)
}

func (u *commentUsecase) GetCommentsByIssueID(
	ctx context.Context,
	issueID uuid.UUID,
) ([]domain.Comment, error) {
	// Verify issue exists
	if _, err := u.issueRepo.GetByID(ctx, issueID); err != nil {
		return nil, err
	}

	return u.commentRepo.ListByIssueID(ctx, issueID)
}

func (u *commentUsecase) UpdateComment(
	ctx context.Context,
	userID, commentID uuid.UUID,
	req domain.UpdateCommentRequest,
) (*domain.Comment, error) {
	content := strings.TrimSpace(req.Content)
	if content == "" {
		return nil, domain.ErrInvalidInput
	}

	comment, err := u.commentRepo.GetByID(ctx, commentID)
	if err != nil {
		return nil, err
	}

	// Only the author can update their own comment
	if comment.UserID != userID {
		return nil, domain.ErrForbidden
	}

	comment.Content = content
	if err := u.commentRepo.Update(ctx, comment); err != nil {
		return nil, err
	}

	return u.commentRepo.GetByID(ctx, comment.ID)
}

func (u *commentUsecase) DeleteComment(
	ctx context.Context,
	userID, commentID uuid.UUID,
) error {
	comment, err := u.commentRepo.GetByID(ctx, commentID)
	if err != nil {
		return err
	}

	// Author can delete their own comment
	if comment.UserID == userID {
		return u.commentRepo.Delete(ctx, commentID)
	}

	// Non-authors must be project OWNER or ADMIN
	if u.rbacRepo != nil && u.issueRepo != nil {
		issue, err := u.issueRepo.GetByID(ctx, comment.IssueID)
		if err == nil && issue != nil {
			member, err := u.rbacRepo.GetUserMemberInProject(ctx, userID, issue.ProjectID)
			if err == nil && member != nil && member.Role != nil {
				if member.Role.Name == domain.RoleOwner || member.Role.Name == domain.RoleAdmin {
					return u.commentRepo.Delete(ctx, commentID)
				}
			}
		}
	}

	return domain.ErrForbidden
}

