package handler

import (
	"errors"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/luqmanulhakimdev/tykket-go/internal/domain"
	"github.com/luqmanulhakimdev/tykket-go/pkg/response"
)

type SprintHandler struct {
	sprintUsecase domain.SprintUsecase
}

func NewSprintHandler(sprintUsecase domain.SprintUsecase) *SprintHandler {
	return &SprintHandler{sprintUsecase: sprintUsecase}
}

func (h *SprintHandler) Create(c echo.Context) error {
	projectID, err := uuid.Parse(c.Param("projectId"))
	if err != nil {
		return response.BadRequest(c, "Invalid project ID", nil)
	}

	var req domain.CreateSprintRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "Invalid request payload", err)
	}
	if err := c.Validate(&req); err != nil {
		return response.BadRequest(c, err.Error(), nil)
	}

	sprint, err := h.sprintUsecase.CreateSprint(c.Request().Context(), projectID, req)
	if err != nil {
		return response.InternalServerError(c, err.Error())
	}

	return response.Created(c, "Sprint created successfully", sprint)
}

func (h *SprintHandler) List(c echo.Context) error {
	projectID, err := uuid.Parse(c.Param("projectId"))
	if err != nil {
		return response.BadRequest(c, "Invalid project ID", nil)
	}

	sprints, err := h.sprintUsecase.ListSprints(c.Request().Context(), projectID)
	if err != nil {
		return response.InternalServerError(c, err.Error())
	}

	return response.OK(c, "Sprints retrieved", sprints)
}

func (h *SprintHandler) GetByID(c echo.Context) error {
	projectID, err := uuid.Parse(c.Param("projectId"))
	if err != nil {
		return response.BadRequest(c, "Invalid project ID", nil)
	}

	sprintID, err := uuid.Parse(c.Param("sprintId"))
	if err != nil {
		return response.BadRequest(c, "Invalid sprint ID", nil)
	}

	sprint, err := h.sprintUsecase.GetSprintByID(c.Request().Context(), projectID, sprintID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return response.NotFound(c, "Sprint not found")
		}
		return response.InternalServerError(c, err.Error())
	}

	return response.OK(c, "Sprint retrieved", sprint)
}

func (h *SprintHandler) Update(c echo.Context) error {
	projectID, err := uuid.Parse(c.Param("projectId"))
	if err != nil {
		return response.BadRequest(c, "Invalid project ID", nil)
	}

	sprintID, err := uuid.Parse(c.Param("sprintId"))
	if err != nil {
		return response.BadRequest(c, "Invalid sprint ID", nil)
	}

	var req domain.UpdateSprintRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "Invalid request payload", err)
	}
	if err := c.Validate(&req); err != nil {
		return response.BadRequest(c, err.Error(), nil)
	}

	sprint, err := h.sprintUsecase.UpdateSprint(c.Request().Context(), projectID, sprintID, req)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return response.NotFound(c, "Sprint not found")
		}
		return response.InternalServerError(c, err.Error())
	}

	return response.OK(c, "Sprint updated successfully", sprint)
}

func (h *SprintHandler) Start(c echo.Context) error {
	projectID, err := uuid.Parse(c.Param("projectId"))
	if err != nil {
		return response.BadRequest(c, "Invalid project ID", nil)
	}

	sprintID, err := uuid.Parse(c.Param("sprintId"))
	if err != nil {
		return response.BadRequest(c, "Invalid sprint ID", nil)
	}

	var req domain.StartSprintRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "Invalid request payload", err)
	}

	sprint, err := h.sprintUsecase.StartSprint(c.Request().Context(), projectID, sprintID, req)
	if err != nil {
		if errors.Is(err, domain.ErrActiveSprintExists) {
			return response.Conflict(c, "An active sprint already exists in this project. Please complete the current active sprint before starting a new one.")
		}
		if errors.Is(err, domain.ErrInvalidInput) {
			return response.BadRequest(c, err.Error(), nil)
		}
		return response.InternalServerError(c, err.Error())
	}

	return response.OK(c, "Sprint started successfully", sprint)
}

func (h *SprintHandler) Complete(c echo.Context) error {
	projectID, err := uuid.Parse(c.Param("projectId"))
	if err != nil {
		return response.BadRequest(c, "Invalid project ID", nil)
	}

	sprintID, err := uuid.Parse(c.Param("sprintId"))
	if err != nil {
		return response.BadRequest(c, "Invalid sprint ID", nil)
	}

	var req domain.CompleteSprintRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "Invalid request payload", err)
	}

	sprint, err := h.sprintUsecase.CompleteSprint(c.Request().Context(), projectID, sprintID, req)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidInput) {
			return response.BadRequest(c, err.Error(), nil)
		}
		return response.InternalServerError(c, err.Error())
	}

	return response.OK(c, "Sprint completed successfully", sprint)
}

func (h *SprintHandler) Delete(c echo.Context) error {
	projectID, err := uuid.Parse(c.Param("projectId"))
	if err != nil {
		return response.BadRequest(c, "Invalid project ID", nil)
	}

	sprintID, err := uuid.Parse(c.Param("sprintId"))
	if err != nil {
		return response.BadRequest(c, "Invalid sprint ID", nil)
	}

	if err := h.sprintUsecase.DeleteSprint(c.Request().Context(), projectID, sprintID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return response.NotFound(c, "Sprint not found")
		}
		return response.InternalServerError(c, err.Error())
	}

	return response.OK(c, "Sprint deleted successfully", nil)
}

func (h *SprintHandler) BulkMove(c echo.Context) error {
	projectID, err := uuid.Parse(c.Param("projectId"))
	if err != nil {
		return response.BadRequest(c, "Invalid project ID", nil)
	}

	var req domain.BulkMoveSprintIssuesRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "Invalid request payload", err)
	}
	if err := c.Validate(&req); err != nil {
		return response.BadRequest(c, err.Error(), nil)
	}

	if err := h.sprintUsecase.BulkMoveIssues(c.Request().Context(), projectID, req); err != nil {
		return response.InternalServerError(c, err.Error())
	}

	return response.OK(c, "Issues moved successfully", nil)
}

