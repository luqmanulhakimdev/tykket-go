package handler

import (
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/luqmanulhakimdev/tykket-go/internal/domain"
	"github.com/luqmanulhakimdev/tykket-go/pkg/response"
)

type MetricHandler struct {
	metricUsecase domain.MetricUsecase
}

func NewMetricHandler(metricUsecase domain.MetricUsecase) *MetricHandler {
	return &MetricHandler{metricUsecase: metricUsecase}
}

func (h *MetricHandler) GetProjectMetrics(c echo.Context) error {
	projectID, err := uuid.Parse(c.Param("projectId"))
	if err != nil {
		return response.BadRequest(c, "Invalid project ID", nil)
	}

	metrics, err := h.metricUsecase.GetProjectMetrics(c.Request().Context(), projectID)
	if err != nil {
		return response.InternalServerError(c, err.Error())
	}

	return response.OK(c, "Project metrics retrieved", metrics)
}

func (h *MetricHandler) GetSprintBurndown(c echo.Context) error {
	projectID, err := uuid.Parse(c.Param("projectId"))
	if err != nil {
		return response.BadRequest(c, "Invalid project ID", nil)
	}

	sprintID, err := uuid.Parse(c.Param("sprintId"))
	if err != nil {
		return response.BadRequest(c, "Invalid sprint ID", nil)
	}

	metrics, err := h.metricUsecase.GetSprintMetrics(c.Request().Context(), projectID, sprintID)
	if err != nil {
		return response.InternalServerError(c, err.Error())
	}

	return response.OK(c, "Sprint metrics retrieved", metrics)
}

func (h *MetricHandler) ExportCSV(c echo.Context) error {
	projectID, err := uuid.Parse(c.Param("projectId"))
	if err != nil {
		return response.BadRequest(c, "Invalid project ID", nil)
	}

	filter := domain.IssueFilter{
		ProjectID: projectID,
	}

	if backlogStr := c.QueryParam("in_backlog"); backlogStr != "" {
		inBacklog := backlogStr == "true" || backlogStr == "1"
		filter.InBacklog = &inBacklog
	} else if sprintIDStr := c.QueryParam("sprint_id"); sprintIDStr != "" {
		if sid, err := uuid.Parse(sprintIDStr); err == nil {
			filter.SprintID = &sid
		}
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

	filename := fmt.Sprintf("issues_export_%s_%d.csv", projectID.String()[:8], time.Now().Unix())

	res := c.Response()
	res.Header().Set(echo.HeaderContentType, "text/csv")
	res.Header().Set(echo.HeaderContentDisposition, fmt.Sprintf("attachment; filename=%q", filename))
	res.WriteHeader(http.StatusOK)

	return h.metricUsecase.ExportIssuesCSV(c.Request().Context(), filter, res)
}

func (h *MetricHandler) ExportSummary(c echo.Context) error {
	projectID, err := uuid.Parse(c.Param("projectId"))
	if err != nil {
		return response.BadRequest(c, "Invalid project ID", nil)
	}

	summary, err := h.metricUsecase.ExportProjectSummary(c.Request().Context(), projectID)
	if err != nil {
		return response.InternalServerError(c, err.Error())
	}

	return response.OK(c, "Project summary export retrieved", summary)
}

