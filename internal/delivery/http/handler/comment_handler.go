package handler

import (
	"errors"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/luqmanulhakimdev/tykket-go/internal/delivery/http/middleware"
	"github.com/luqmanulhakimdev/tykket-go/internal/domain"
	"github.com/luqmanulhakimdev/tykket-go/pkg/response"
)

type CommentHandler struct {
	commentUsecase domain.CommentUsecase
	issueRepo      domain.IssueRepository
}

func NewCommentHandler(
	commentUsecase domain.CommentUsecase,
	issueRepo domain.IssueRepository,
) *CommentHandler {
	return &CommentHandler{
		commentUsecase: commentUsecase,
		issueRepo:      issueRepo,
	}
}

func (h *CommentHandler) resolveIssueID(c echo.Context) (uuid.UUID, error) {
	param := c.Param("issueId")
	if param == "" {
		param = c.Param("id")
	}
	if param == "" {
		return uuid.Nil, domain.ErrInvalidInput
	}

	id, err := uuid.Parse(param)
	if err == nil {
		return id, nil
	}

	if h.issueRepo != nil {
		issue, err := h.issueRepo.GetByKey(c.Request().Context(), param)
		if err == nil && issue != nil {
			return issue.ID, nil
		}
	}

	return uuid.Nil, domain.ErrNotFound
}

func (h *CommentHandler) ListByIssue(c echo.Context) error {
	issueID, err := h.resolveIssueID(c)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return response.NotFound(c, "Issue not found")
		}
		return response.BadRequest(c, "Invalid issue identifier", err)
	}

	comments, err := h.commentUsecase.GetCommentsByIssueID(c.Request().Context(), issueID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return response.NotFound(c, "Issue not found")
		}
		return response.InternalServerError(c, err.Error())
	}

	return response.OK(c, "Comments retrieved successfully", comments)
}

func (h *CommentHandler) Create(c echo.Context) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return response.Unauthorized(c, "Unauthorized")
	}

	issueID, err := h.resolveIssueID(c)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return response.NotFound(c, "Issue not found")
		}
		return response.BadRequest(c, "Invalid issue identifier", err)
	}

	var req domain.CreateCommentRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "Invalid request payload", err)
	}
	if err := c.Validate(&req); err != nil {
		return response.BadRequest(c, err.Error(), nil)
	}

	comment, err := h.commentUsecase.CreateComment(c.Request().Context(), userID, issueID, req)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return response.NotFound(c, "Issue not found")
		}
		if errors.Is(err, domain.ErrInvalidInput) {
			return response.BadRequest(c, err.Error(), nil)
		}
		return response.InternalServerError(c, err.Error())
	}

	return response.Created(c, "Comment created successfully", comment)
}

func (h *CommentHandler) Update(c echo.Context) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return response.Unauthorized(c, "Unauthorized")
	}

	commentID, err := uuid.Parse(c.Param("commentId"))
	if err != nil {
		return response.BadRequest(c, "Invalid comment ID", err)
	}

	var req domain.UpdateCommentRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "Invalid request payload", err)
	}
	if err := c.Validate(&req); err != nil {
		return response.BadRequest(c, err.Error(), nil)
	}

	comment, err := h.commentUsecase.UpdateComment(c.Request().Context(), userID, commentID, req)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return response.NotFound(c, "Comment not found")
		}
		if errors.Is(err, domain.ErrForbidden) {
			return response.Forbidden(c, "You can only edit your own comment")
		}
		if errors.Is(err, domain.ErrInvalidInput) {
			return response.BadRequest(c, err.Error(), nil)
		}
		return response.InternalServerError(c, err.Error())
	}

	return response.OK(c, "Comment updated successfully", comment)
}

func (h *CommentHandler) Delete(c echo.Context) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return response.Unauthorized(c, "Unauthorized")
	}

	commentID, err := uuid.Parse(c.Param("commentId"))
	if err != nil {
		return response.BadRequest(c, "Invalid comment ID", err)
	}

	err = h.commentUsecase.DeleteComment(c.Request().Context(), userID, commentID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return response.NotFound(c, "Comment not found")
		}
		if errors.Is(err, domain.ErrForbidden) {
			return response.Forbidden(c, "You do not have permission to delete this comment")
		}
		return response.InternalServerError(c, err.Error())
	}

	return response.OK(c, "Comment deleted successfully", nil)
}

