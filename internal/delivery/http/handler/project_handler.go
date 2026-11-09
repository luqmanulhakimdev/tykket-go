package handler

import (
	"errors"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/luqmanulhakimdev/tykket-go/internal/delivery/http/middleware"
	"github.com/luqmanulhakimdev/tykket-go/internal/domain"
	"github.com/luqmanulhakimdev/tykket-go/pkg/response"
)

type ProjectHandler struct {
	projectUsecase domain.ProjectUsecase
}

func NewProjectHandler(projectUsecase domain.ProjectUsecase) *ProjectHandler {
	return &ProjectHandler{projectUsecase: projectUsecase}
}

func (h *ProjectHandler) Create(c echo.Context) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return response.Unauthorized(c, "Unauthorized")
	}

	var req domain.CreateProjectRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "Invalid request payload", err)
	}
	if err := c.Validate(&req); err != nil {
		return response.BadRequest(c, err.Error(), nil)
	}

	project, err := h.projectUsecase.CreateProject(c.Request().Context(), userID, req)
	if err != nil {
		if errors.Is(err, domain.ErrConflict) {
			return response.Conflict(c, err.Error())
		}
		if errors.Is(err, domain.ErrInvalidInput) {
			return response.BadRequest(c, err.Error(), nil)
		}
		return response.InternalServerError(c, err.Error())
	}

	return response.Created(c, "Project created successfully", project)
}

func (h *ProjectHandler) List(c echo.Context) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return response.Unauthorized(c, "Unauthorized")
	}

	projects, err := h.projectUsecase.ListProjects(c.Request().Context(), userID)
	if err != nil {
		return response.InternalServerError(c, err.Error())
	}

	return response.OK(c, "Projects retrieved", projects)
}

func (h *ProjectHandler) GetByID(c echo.Context) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return response.Unauthorized(c, "Unauthorized")
	}

	projectID, err := uuid.Parse(c.Param("projectId"))
	if err != nil {
		return response.BadRequest(c, "Invalid project ID", nil)
	}

	project, err := h.projectUsecase.GetProjectByID(c.Request().Context(), userID, projectID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return response.NotFound(c, "Project not found")
		}
		return response.InternalServerError(c, err.Error())
	}

	return response.OK(c, "Project retrieved", project)
}

func (h *ProjectHandler) Update(c echo.Context) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return response.Unauthorized(c, "Unauthorized")
	}

	projectID, err := uuid.Parse(c.Param("projectId"))
	if err != nil {
		return response.BadRequest(c, "Invalid project ID", nil)
	}

	var req domain.UpdateProjectRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "Invalid request payload", err)
	}
	if err := c.Validate(&req); err != nil {
		return response.BadRequest(c, err.Error(), nil)
	}

	project, err := h.projectUsecase.UpdateProject(c.Request().Context(), userID, projectID, req)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return response.NotFound(c, "Project not found")
		}
		return response.InternalServerError(c, err.Error())
	}

	return response.OK(c, "Project updated successfully", project)
}

func (h *ProjectHandler) Delete(c echo.Context) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return response.Unauthorized(c, "Unauthorized")
	}

	projectID, err := uuid.Parse(c.Param("projectId"))
	if err != nil {
		return response.BadRequest(c, "Invalid project ID", nil)
	}

	if err := h.projectUsecase.DeleteProject(c.Request().Context(), userID, projectID); err != nil {
		if errors.Is(err, domain.ErrForbidden) {
			return response.Forbidden(c, "Only the project owner can delete this project")
		}
		if errors.Is(err, domain.ErrNotFound) {
			return response.NotFound(c, "Project not found")
		}
		return response.InternalServerError(c, err.Error())
	}

	return response.OK(c, "Project deleted successfully", nil)
}

func (h *ProjectHandler) ListMembers(c echo.Context) error {
	projectID, err := uuid.Parse(c.Param("projectId"))
	if err != nil {
		return response.BadRequest(c, "Invalid project ID", nil)
	}

	members, err := h.projectUsecase.ListMembers(c.Request().Context(), projectID)
	if err != nil {
		return response.InternalServerError(c, err.Error())
	}

	return response.OK(c, "Project members retrieved", members)
}

func (h *ProjectHandler) AddMember(c echo.Context) error {
	projectID, err := uuid.Parse(c.Param("projectId"))
	if err != nil {
		return response.BadRequest(c, "Invalid project ID", nil)
	}

	var req domain.AddMemberRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "Invalid request payload", err)
	}
	if err := c.Validate(&req); err != nil {
		return response.BadRequest(c, err.Error(), nil)
	}

	member, err := h.projectUsecase.AddMember(c.Request().Context(), projectID, req)
	if err != nil {
		if errors.Is(err, domain.ErrConflict) {
			return response.Conflict(c, "User is already a member of this project")
		}
		if errors.Is(err, domain.ErrNotFound) {
			return response.NotFound(c, err.Error())
		}
		return response.InternalServerError(c, err.Error())
	}

	return response.Created(c, "Member added successfully", member)
}

func (h *ProjectHandler) UpdateMemberRole(c echo.Context) error {
	projectID, err := uuid.Parse(c.Param("projectId"))
	if err != nil {
		return response.BadRequest(c, "Invalid project ID", nil)
	}

	targetUserID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		return response.BadRequest(c, "Invalid target user ID", nil)
	}

	var req domain.UpdateMemberRoleRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "Invalid request payload", err)
	}
	if err := c.Validate(&req); err != nil {
		return response.BadRequest(c, err.Error(), nil)
	}

	member, err := h.projectUsecase.UpdateMemberRole(c.Request().Context(), projectID, targetUserID, req)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return response.NotFound(c, err.Error())
		}
		return response.InternalServerError(c, err.Error())
	}

	return response.OK(c, "Member role updated successfully", member)
}

func (h *ProjectHandler) RemoveMember(c echo.Context) error {
	projectID, err := uuid.Parse(c.Param("projectId"))
	if err != nil {
		return response.BadRequest(c, "Invalid project ID", nil)
	}

	targetUserID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		return response.BadRequest(c, "Invalid target user ID", nil)
	}

	if err := h.projectUsecase.RemoveMember(c.Request().Context(), projectID, targetUserID); err != nil {
		if errors.Is(err, domain.ErrForbidden) {
			return response.Forbidden(c, err.Error())
		}
		if errors.Is(err, domain.ErrNotFound) {
			return response.NotFound(c, "Member not found")
		}
		return response.InternalServerError(c, err.Error())
	}

	return response.OK(c, "Member removed successfully", nil)
}

