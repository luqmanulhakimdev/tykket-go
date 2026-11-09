package handler

import (
	"errors"
	"strconv"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/luqmanulhakimdev/tykket-go/internal/delivery/http/middleware"
	"github.com/luqmanulhakimdev/tykket-go/internal/domain"
	"github.com/luqmanulhakimdev/tykket-go/pkg/response"
)

type IssueHandler struct {
	issueUsecase domain.IssueUsecase
}

func NewIssueHandler(issueUsecase domain.IssueUsecase) *IssueHandler {
	return &IssueHandler{issueUsecase: issueUsecase}
}

func (h *IssueHandler) Create(c echo.Context) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return response.Unauthorized(c, "Unauthorized")
	}

	projectID, err := uuid.Parse(c.Param("projectId"))
	if err != nil {
		return response.BadRequest(c, "Invalid project ID", nil)
	}

	var req domain.CreateIssueRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "Invalid request payload", err)
	}
	if err := c.Validate(&req); err != nil {
		return response.BadRequest(c, err.Error(), nil)
	}

	issue, err := h.issueUsecase.CreateIssue(c.Request().Context(), userID, projectID, req)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidHierarchy) || errors.Is(err, domain.ErrInvalidInput) {
			return response.BadRequest(c, err.Error(), nil)
		}
		if errors.Is(err, domain.ErrNotFound) {
			return response.NotFound(c, err.Error())
		}
		return response.InternalServerError(c, err.Error())
	}

	return response.Created(c, "Issue created successfully", issue)
}

func (h *IssueHandler) List(c echo.Context) error {
	projectID, err := uuid.Parse(c.Param("projectId"))
	if err != nil {
		return response.BadRequest(c, "Invalid project ID", nil)
	}

	filter := domain.IssueFilter{
		ProjectID: projectID,
		Search:    c.QueryParam("search"),
		SortBy:    c.QueryParam("sort_by"),
		SortOrder: c.QueryParam("sort_order"),
	}

	if backlogStr := c.QueryParam("in_backlog"); backlogStr != "" {
		inBacklog := backlogStr == "true" || backlogStr == "1"
		filter.InBacklog = &inBacklog
	} else if sprintIDStr := c.QueryParam("sprint_id"); sprintIDStr != "" {
		if sid, err := uuid.Parse(sprintIDStr); err == nil {
			filter.SprintID = &sid
		}
	}

	if parentIDStr := c.QueryParam("parent_id"); parentIDStr != "" {
		if pid, err := uuid.Parse(parentIDStr); err == nil {
			filter.ParentID = &pid
		}
	}

	if typeStr := c.QueryParam("type"); typeStr != "" {
		t := domain.IssueType(typeStr)
		filter.Type = &t
	}

	if statusStr := c.QueryParam("status"); statusStr != "" {
		s := domain.IssueStatus(statusStr)
		filter.Status = &s
	}

	if assigneeStr := c.QueryParam("assignee_id"); assigneeStr != "" {
		if aid, err := uuid.Parse(assigneeStr); err == nil {
			filter.AssigneeID = &aid
		}
	}

	page := 1
	limit := 50
	if p, err := strconv.Atoi(c.QueryParam("page")); err == nil && p > 0 {
		page = p
	}
	if l, err := strconv.Atoi(c.QueryParam("limit")); err == nil && l > 0 && l <= 100 {
		limit = l
	}

	filter.Limit = limit
	filter.Offset = (page - 1) * limit

	issues, total, err := h.issueUsecase.ListIssues(c.Request().Context(), filter)
	if err != nil {
		return response.InternalServerError(c, err.Error())
	}

	totalPages := int((total + int64(limit) - 1) / int64(limit))

	return response.OKWithMeta(c, "Issues retrieved", issues, &response.Meta{
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
	})
}

func (h *IssueHandler) GetByID(c echo.Context) error {
	projectID, err := uuid.Parse(c.Param("projectId"))
	if err != nil {
		return response.BadRequest(c, "Invalid project ID", nil)
	}

	issueID, err := uuid.Parse(c.Param("issueId"))
	if err != nil {
		return response.BadRequest(c, "Invalid issue ID", nil)
	}

	issue, err := h.issueUsecase.GetIssueByID(c.Request().Context(), projectID, issueID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return response.NotFound(c, "Issue not found")
		}
		return response.InternalServerError(c, err.Error())
	}

	return response.OK(c, "Issue retrieved", issue)
}

func (h *IssueHandler) GetByKey(c echo.Context) error {
	key := c.Param("key")
	if key == "" {
		return response.BadRequest(c, "Issue key is required", nil)
	}

	issue, err := h.issueUsecase.GetIssueByKey(c.Request().Context(), key)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return response.NotFound(c, "Issue not found")
		}
		return response.InternalServerError(c, err.Error())
	}

	return response.OK(c, "Issue retrieved", issue)
}

func (h *IssueHandler) Update(c echo.Context) error {
	projectID, err := uuid.Parse(c.Param("projectId"))
	if err != nil {
		return response.BadRequest(c, "Invalid project ID", nil)
	}

	issueID, err := uuid.Parse(c.Param("issueId"))
	if err != nil {
		return response.BadRequest(c, "Invalid issue ID", nil)
	}

	var req domain.UpdateIssueRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "Invalid request payload", err)
	}
	if err := c.Validate(&req); err != nil {
		return response.BadRequest(c, err.Error(), nil)
	}

	issue, err := h.issueUsecase.UpdateIssue(c.Request().Context(), projectID, issueID, req)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidHierarchy) {
			return response.BadRequest(c, err.Error(), nil)
		}
		if errors.Is(err, domain.ErrNotFound) {
			return response.NotFound(c, "Issue not found")
		}
		return response.InternalServerError(c, err.Error())
	}

	return response.OK(c, "Issue updated successfully", issue)
}

func (h *IssueHandler) Move(c echo.Context) error {
	projectID, err := uuid.Parse(c.Param("projectId"))
	if err != nil {
		return response.BadRequest(c, "Invalid project ID", nil)
	}

	issueID, err := uuid.Parse(c.Param("issueId"))
	if err != nil {
		return response.BadRequest(c, "Invalid issue ID", nil)
	}

	var req domain.MoveIssueRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "Invalid request payload", err)
	}
	if err := c.Validate(&req); err != nil {
		return response.BadRequest(c, err.Error(), nil)
	}

	issue, err := h.issueUsecase.MoveIssue(c.Request().Context(), projectID, issueID, req)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return response.NotFound(c, "Issue not found")
		}
		return response.InternalServerError(c, err.Error())
	}

	return response.OK(c, "Issue moved successfully", issue)
}

func (h *IssueHandler) Delete(c echo.Context) error {
	projectID, err := uuid.Parse(c.Param("projectId"))
	if err != nil {
		return response.BadRequest(c, "Invalid project ID", nil)
	}

	issueID, err := uuid.Parse(c.Param("issueId"))
	if err != nil {
		return response.BadRequest(c, "Invalid issue ID", nil)
	}

	if err := h.issueUsecase.DeleteIssue(c.Request().Context(), projectID, issueID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return response.NotFound(c, "Issue not found")
		}
		return response.InternalServerError(c, err.Error())
	}

	return response.OK(c, "Issue deleted successfully", nil)
}

