package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/luqmanulhakimdev/tykket-go/internal/delivery/http/middleware"
	"github.com/luqmanulhakimdev/tykket-go/internal/domain"
)

type fullMockProjectUC struct {
	domain.ProjectUsecase
	err error
}

func (m *fullMockProjectUC) CreateProject(ctx context.Context, userID uuid.UUID, req domain.CreateProjectRequest) (*domain.Project, error) {
	return nil, m.err
}
func (m *fullMockProjectUC) GetProjectByID(ctx context.Context, userID, projectID uuid.UUID) (*domain.Project, error) {
	return nil, m.err
}
func (m *fullMockProjectUC) ListProjects(ctx context.Context, userID uuid.UUID) ([]domain.Project, error) {
	return nil, m.err
}
func (m *fullMockProjectUC) UpdateProject(ctx context.Context, userID, projectID uuid.UUID, req domain.UpdateProjectRequest) (*domain.Project, error) {
	return nil, m.err
}
func (m *fullMockProjectUC) DeleteProject(ctx context.Context, userID, projectID uuid.UUID) error {
	return m.err
}
func (m *fullMockProjectUC) ListMembers(ctx context.Context, projectID uuid.UUID) ([]domain.ProjectMember, error) {
	return nil, m.err
}
func (m *fullMockProjectUC) AddMember(ctx context.Context, projectID uuid.UUID, req domain.AddMemberRequest) (*domain.ProjectMember, error) {
	return nil, m.err
}
func (m *fullMockProjectUC) UpdateMemberRole(ctx context.Context, projectID, targetUserID uuid.UUID, req domain.UpdateMemberRoleRequest) (*domain.ProjectMember, error) {
	return nil, m.err
}
func (m *fullMockProjectUC) RemoveMember(ctx context.Context, projectID, targetUserID uuid.UUID) error {
	return m.err
}

type fullMockSprintUC struct {
	domain.SprintUsecase
	err error
}

func (m *fullMockSprintUC) CreateSprint(ctx context.Context, projectID uuid.UUID, req domain.CreateSprintRequest) (*domain.Sprint, error) {
	return nil, m.err
}
func (m *fullMockSprintUC) GetSprintByID(ctx context.Context, projectID, sprintID uuid.UUID) (*domain.Sprint, error) {
	return nil, m.err
}
func (m *fullMockSprintUC) ListSprints(ctx context.Context, projectID uuid.UUID) ([]domain.Sprint, error) {
	return nil, m.err
}
func (m *fullMockSprintUC) UpdateSprint(ctx context.Context, projectID, sprintID uuid.UUID, req domain.UpdateSprintRequest) (*domain.Sprint, error) {
	return nil, m.err
}
func (m *fullMockSprintUC) StartSprint(ctx context.Context, projectID, sprintID uuid.UUID, req domain.StartSprintRequest) (*domain.Sprint, error) {
	return nil, m.err
}
func (m *fullMockSprintUC) CompleteSprint(ctx context.Context, projectID, sprintID uuid.UUID, req domain.CompleteSprintRequest) (*domain.Sprint, error) {
	return nil, m.err
}
func (m *fullMockSprintUC) DeleteSprint(ctx context.Context, projectID, sprintID uuid.UUID) error {
	return m.err
}
func (m *fullMockSprintUC) BulkMoveIssues(ctx context.Context, projectID uuid.UUID, req domain.BulkMoveSprintIssuesRequest) error {
	return m.err
}

type fullMockIssueUC struct {
	domain.IssueUsecase
	err error
}

func (m *fullMockIssueUC) CreateIssue(ctx context.Context, reporterID, projectID uuid.UUID, req domain.CreateIssueRequest) (*domain.Issue, error) {
	return nil, m.err
}
func (m *fullMockIssueUC) GetIssueByID(ctx context.Context, projectID, issueID uuid.UUID) (*domain.Issue, error) {
	return nil, m.err
}
func (m *fullMockIssueUC) GetIssueByKey(ctx context.Context, key string) (*domain.Issue, error) {
	return nil, m.err
}
func (m *fullMockIssueUC) ListIssues(ctx context.Context, filter domain.IssueFilter) ([]domain.Issue, int64, error) {
	return nil, 0, m.err
}
func (m *fullMockIssueUC) UpdateIssue(ctx context.Context, projectID, issueID uuid.UUID, req domain.UpdateIssueRequest) (*domain.Issue, error) {
	return nil, m.err
}
func (m *fullMockIssueUC) MoveIssue(ctx context.Context, projectID, issueID uuid.UUID, req domain.MoveIssueRequest) (*domain.Issue, error) {
	return nil, m.err
}
func (m *fullMockIssueUC) DeleteIssue(ctx context.Context, projectID, issueID uuid.UUID) error {
	return m.err
}

func TestHandlers_ExhaustiveBranchCoverage(t *testing.T) {
	e := setupEcho()
	validID := uuid.New()
	targetID := uuid.New()

	// 1. ProjectHandler Exhaustive
	t.Run("ProjectHandler Exhaustive", func(t *testing.T) {
		// NotFound on UpdateMemberRole
		hNF := NewProjectHandler(&fullMockProjectUC{err: domain.ErrNotFound})
		reqBody, _ := json.Marshal(domain.UpdateMemberRoleRequest{RoleName: "ADMIN"})
		req := httptest.NewRequest(http.MethodPut, "/", bytes.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("projectId", "userId")
		c.SetParamValues(validID.String(), targetID.String())
		_ = hNF.UpdateMemberRole(c)
		if rec.Code != http.StatusNotFound {
			t.Errorf("expected 404, got %d", rec.Code)
		}

		// Validation error on UpdateMemberRole
		reqBadVal := httptest.NewRequest(http.MethodPut, "/", bytes.NewReader([]byte(`{"role_name":"SUPERUSER"}`)))
		reqBadVal.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		recBadVal := httptest.NewRecorder()
		cBadVal := e.NewContext(reqBadVal, recBadVal)
		cBadVal.SetParamNames("projectId", "userId")
		cBadVal.SetParamValues(validID.String(), targetID.String())
		_ = hNF.UpdateMemberRole(cBadVal)
		if recBadVal.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for bad role validation, got %d", recBadVal.Code)
		}

		// RemoveMember NotFound
		recRemNF := httptest.NewRecorder()
		cRemNF := e.NewContext(req, recRemNF)
		cRemNF.SetParamNames("projectId", "userId")
		cRemNF.SetParamValues(validID.String(), targetID.String())
		_ = hNF.RemoveMember(cRemNF)
		if recRemNF.Code != http.StatusNotFound {
			t.Errorf("expected 404, got %d", recRemNF.Code)
		}

		// RemoveMember InternalServerError
		hErr := NewProjectHandler(&fullMockProjectUC{err: errors.New("server err")})
		recRemErr := httptest.NewRecorder()
		cRemErr := e.NewContext(req, recRemErr)
		cRemErr.SetParamNames("projectId", "userId")
		cRemErr.SetParamValues(validID.String(), targetID.String())
		_ = hErr.RemoveMember(cRemErr)
		if recRemErr.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", recRemErr.Code)
		}

		// ListMembers bad project ID
		recLM := httptest.NewRecorder()
		cLM := e.NewContext(req, recLM)
		cLM.SetParamNames("projectId")
		cLM.SetParamValues("bad")
		_ = hNF.ListMembers(cLM)
		if recLM.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", recLM.Code)
		}
	})

	// 2. SprintHandler Exhaustive
	t.Run("SprintHandler Exhaustive", func(t *testing.T) {
		hNF := NewSprintHandler(&fullMockSprintUC{err: domain.ErrNotFound})
		hErr := NewSprintHandler(&fullMockSprintUC{err: errors.New("server err")})

		reqBody, _ := json.Marshal(domain.UpdateSprintRequest{Name: "Sprint Updated"})
		req := httptest.NewRequest(http.MethodPut, "/", bytes.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

		// GetByID NotFound and InternalError
		recGetNF := httptest.NewRecorder()
		cGetNF := e.NewContext(req, recGetNF)
		cGetNF.SetParamNames("projectId", "sprintId")
		cGetNF.SetParamValues(validID.String(), targetID.String())
		_ = hNF.GetByID(cGetNF)
		if recGetNF.Code != http.StatusNotFound {
			t.Errorf("expected 404, got %d", recGetNF.Code)
		}

		recGetErr := httptest.NewRecorder()
		cGetErr := e.NewContext(req, recGetErr)
		cGetErr.SetParamNames("projectId", "sprintId")
		cGetErr.SetParamValues(validID.String(), targetID.String())
		_ = hErr.GetByID(cGetErr)
		if recGetErr.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", recGetErr.Code)
		}

		// Update NotFound and InternalError
		recUpdNF := httptest.NewRecorder()
		cUpdNF := e.NewContext(req, recUpdNF)
		cUpdNF.SetParamNames("projectId", "sprintId")
		cUpdNF.SetParamValues(validID.String(), targetID.String())
		_ = hNF.Update(cUpdNF)
		if recUpdNF.Code != http.StatusNotFound {
			t.Errorf("expected 404, got %d", recUpdNF.Code)
		}

		recUpdErr := httptest.NewRecorder()
		reqUpdErr := httptest.NewRequest(http.MethodPut, "/", bytes.NewReader(reqBody))
		reqUpdErr.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		cUpdErr := e.NewContext(reqUpdErr, recUpdErr)
		cUpdErr.SetParamNames("projectId", "sprintId")
		cUpdErr.SetParamValues(validID.String(), targetID.String())
		_ = hErr.Update(cUpdErr)
		if recUpdErr.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", recUpdErr.Code)
		}

		// Update Sprint bad request / validation
		reqBadUpd := httptest.NewRequest(http.MethodPut, "/", bytes.NewReader([]byte(`{"name":""}`)))
		reqBadUpd.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		recBadUpd := httptest.NewRecorder()
		cBadUpd := e.NewContext(reqBadUpd, recBadUpd)
		cBadUpd.SetParamNames("projectId", "sprintId")
		cBadUpd.SetParamValues(validID.String(), targetID.String())
		_ = hNF.Update(cBadUpd)
		if recBadUpd.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for empty name validation, got %d", recBadUpd.Code)
		}

		// Start / Complete bad sprint ID & Internal error
		recStartBad := httptest.NewRecorder()
		cStartBad := e.NewContext(httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{}`)), recStartBad)
		cStartBad.Request().Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		cStartBad.SetParamNames("projectId", "sprintId")
		cStartBad.SetParamValues(validID.String(), "bad")
		_ = hErr.Start(cStartBad)
		if recStartBad.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", recStartBad.Code)
		}

		recStartErr := httptest.NewRecorder()
		cStartErr := e.NewContext(httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{}`)), recStartErr)
		cStartErr.Request().Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		cStartErr.SetParamNames("projectId", "sprintId")
		cStartErr.SetParamValues(validID.String(), targetID.String())
		_ = hErr.Start(cStartErr)
		if recStartErr.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", recStartErr.Code)
		}

		recCompBad := httptest.NewRecorder()
		cCompBad := e.NewContext(httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{}`)), recCompBad)
		cCompBad.Request().Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		cCompBad.SetParamNames("projectId", "sprintId")
		cCompBad.SetParamValues(validID.String(), "bad")
		_ = hErr.Complete(cCompBad)
		if recCompBad.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", recCompBad.Code)
		}

		recCompErr := httptest.NewRecorder()
		cCompErr := e.NewContext(httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{}`)), recCompErr)
		cCompErr.Request().Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		cCompErr.SetParamNames("projectId", "sprintId")
		cCompErr.SetParamValues(validID.String(), targetID.String())
		_ = hErr.Complete(cCompErr)
		if recCompErr.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", recCompErr.Code)
		}

		// Delete internal error
		recDelErr := httptest.NewRecorder()
		cDelErr := e.NewContext(req, recDelErr)
		cDelErr.SetParamNames("projectId", "sprintId")
		cDelErr.SetParamValues(validID.String(), targetID.String())
		_ = hErr.Delete(cDelErr)
		if recDelErr.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", recDelErr.Code)
		}

		// BulkMove internal error
		reqBulk, _ := json.Marshal(domain.BulkMoveSprintIssuesRequest{IssueIDs: []uuid.UUID{validID}})
		recBMErr := httptest.NewRecorder()
		cBMErr := e.NewContext(httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(reqBulk)), recBMErr)
		cBMErr.Request().Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		cBMErr.SetParamNames("projectId")
		cBMErr.SetParamValues(validID.String())
		_ = hErr.BulkMove(cBMErr)
		if recBMErr.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", recBMErr.Code)
		}
	})

	// 3. IssueHandler Exhaustive
	t.Run("IssueHandler Exhaustive", func(t *testing.T) {
		hNF := NewIssueHandler(&fullMockIssueUC{err: domain.ErrNotFound})
		hHier := NewIssueHandler(&fullMockIssueUC{err: domain.ErrInvalidHierarchy})
		hErr := NewIssueHandler(&fullMockIssueUC{err: errors.New("server err")})

		reqList := httptest.NewRequest(http.MethodGet, "/", nil)

		// Create Issue bad project ID
		recCrBad := httptest.NewRecorder()
		cCrBad := e.NewContext(reqList, recCrBad)
		cCrBad.Set(middleware.CtxKeyUserID, validID)
		cCrBad.SetParamNames("projectId")
		cCrBad.SetParamValues("bad")
		_ = hNF.Create(cCrBad)
		if recCrBad.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for bad project id, got %d", recCrBad.Code)
		}

		// GetByID bad project ID, bad issue ID, internal error
		recGBadP := httptest.NewRecorder()
		cGBadP := e.NewContext(reqList, recGBadP)
		cGBadP.SetParamNames("projectId", "issueId")
		cGBadP.SetParamValues("bad", targetID.String())
		_ = hNF.GetByID(cGBadP)
		if recGBadP.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for bad project id, got %d", recGBadP.Code)
		}

		recGBadI := httptest.NewRecorder()
		cGBadI := e.NewContext(reqList, recGBadI)
		cGBadI.SetParamNames("projectId", "issueId")
		cGBadI.SetParamValues(validID.String(), "bad")
		_ = hNF.GetByID(cGBadI)
		if recGBadI.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for bad issue id, got %d", recGBadI.Code)
		}

		recGErr := httptest.NewRecorder()
		cGErr := e.NewContext(reqList, recGErr)
		cGErr.SetParamNames("projectId", "issueId")
		cGErr.SetParamValues(validID.String(), targetID.String())
		_ = hErr.GetByID(cGErr)
		if recGErr.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", recGErr.Code)
		}

		// GetByKey empty key and internal error
		recKeyEmpty := httptest.NewRecorder()
		cKeyEmpty := e.NewContext(reqList, recKeyEmpty)
		_ = hNF.GetByKey(cKeyEmpty)
		if recKeyEmpty.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for empty key, got %d", recKeyEmpty.Code)
		}

		recKeyErr := httptest.NewRecorder()
		cKeyErr := e.NewContext(reqList, recKeyErr)
		cKeyErr.SetParamNames("key")
		cKeyErr.SetParamValues("TYK-1")
		_ = hErr.GetByKey(cKeyErr)
		if recKeyErr.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", recKeyErr.Code)
		}

		// Update Issue bad project ID, bad issue ID, validation error, hierarchy error, internal error
		recUBadP := httptest.NewRecorder()
		cUBadP := e.NewContext(reqList, recUBadP)
		cUBadP.SetParamNames("projectId", "issueId")
		cUBadP.SetParamValues("bad", targetID.String())
		_ = hNF.Update(cUBadP)
		if recUBadP.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", recUBadP.Code)
		}

		recUBadI := httptest.NewRecorder()
		cUBadI := e.NewContext(reqList, recUBadI)
		cUBadI.SetParamNames("projectId", "issueId")
		cUBadI.SetParamValues(validID.String(), "bad")
		_ = hNF.Update(cUBadI)
		if recUBadI.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", recUBadI.Code)
		}

		updBody, _ := json.Marshal(domain.UpdateIssueRequest{})
		recUHier := httptest.NewRecorder()
		reqUHier := httptest.NewRequest(http.MethodPut, "/", bytes.NewReader(updBody))
		reqUHier.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		cUHier := e.NewContext(reqUHier, recUHier)
		cUHier.SetParamNames("projectId", "issueId")
		cUHier.SetParamValues(validID.String(), targetID.String())
		_ = hHier.Update(cUHier)
		if recUHier.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for hierarchy error, got %d", recUHier.Code)
		}

		recUErr := httptest.NewRecorder()
		reqUErr := httptest.NewRequest(http.MethodPut, "/", bytes.NewReader(updBody))
		reqUErr.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		cUErr := e.NewContext(reqUErr, recUErr)
		cUErr.SetParamNames("projectId", "issueId")
		cUErr.SetParamValues(validID.String(), targetID.String())
		_ = hErr.Update(cUErr)
		if recUErr.Code != http.StatusInternalServerError {
			t.Errorf("expected 500 for internal error, got %d", recUErr.Code)
		}

		// Move Issue bad project ID, bad issue ID, internal error
		moveBody, _ := json.Marshal(domain.MoveIssueRequest{Position: 100})
		recMBadP := httptest.NewRecorder()
		cMBadP := e.NewContext(reqList, recMBadP)
		cMBadP.SetParamNames("projectId", "issueId")
		cMBadP.SetParamValues("bad", targetID.String())
		_ = hNF.Move(cMBadP)
		if recMBadP.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", recMBadP.Code)
		}

		recMBadI := httptest.NewRecorder()
		cMBadI := e.NewContext(reqList, recMBadI)
		cMBadI.SetParamNames("projectId", "issueId")
		cMBadI.SetParamValues(validID.String(), "bad")
		_ = hNF.Move(cMBadI)
		if recMBadI.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", recMBadI.Code)
		}

		recMErr := httptest.NewRecorder()
		reqMErr := httptest.NewRequest(http.MethodPatch, "/", bytes.NewReader(moveBody))
		reqMErr.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		cMErr := e.NewContext(reqMErr, recMErr)
		cMErr.SetParamNames("projectId", "issueId")
		cMErr.SetParamValues(validID.String(), targetID.String())
		_ = hErr.Move(cMErr)
		if recMErr.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", recMErr.Code)
		}

		// Delete Issue bad project ID, bad issue ID, internal error
		recDBadP := httptest.NewRecorder()
		cDBadP := e.NewContext(reqList, recDBadP)
		cDBadP.SetParamNames("projectId", "issueId")
		cDBadP.SetParamValues("bad", targetID.String())
		_ = hNF.Delete(cDBadP)
		if recDBadP.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", recDBadP.Code)
		}

		recDBadI := httptest.NewRecorder()
		cDBadI := e.NewContext(reqList, recDBadI)
		cDBadI.SetParamNames("projectId", "issueId")
		cDBadI.SetParamValues(validID.String(), "bad")
		_ = hNF.Delete(cDBadI)
		if recDBadI.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", recDBadI.Code)
		}

		recDErr := httptest.NewRecorder()
		cDErr := e.NewContext(reqList, recDErr)
		cDErr.SetParamNames("projectId", "issueId")
		cDErr.SetParamValues(validID.String(), targetID.String())
		_ = hErr.Delete(cDErr)
		if recDErr.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", recDErr.Code)
		}
	})

	// 4. MetricHandler Exhaustive
	t.Run("MetricHandler Exhaustive", func(t *testing.T) {
		h := NewMetricHandler(&mockMetricUsecaseForHandler{})
		reqList := httptest.NewRequest(http.MethodGet, "/", nil)

		// GetProjectMetrics bad project ID
		recPMBad := httptest.NewRecorder()
		cPMBad := e.NewContext(reqList, recPMBad)
		cPMBad.SetParamNames("projectId")
		cPMBad.SetParamValues("bad")
		_ = h.GetProjectMetrics(cPMBad)
		if recPMBad.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", recPMBad.Code)
		}

		// GetSprintBurndown bad project ID, bad sprint ID
		recSBBadP := httptest.NewRecorder()
		cSBBadP := e.NewContext(reqList, recSBBadP)
		cSBBadP.SetParamNames("projectId", "sprintId")
		cSBBadP.SetParamValues("bad", targetID.String())
		_ = h.GetSprintBurndown(cSBBadP)
		if recSBBadP.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", recSBBadP.Code)
		}

		recSBBadS := httptest.NewRecorder()
		cSBBadS := e.NewContext(reqList, recSBBadS)
		cSBBadS.SetParamNames("projectId", "sprintId")
		cSBBadS.SetParamValues(validID.String(), "bad")
		_ = h.GetSprintBurndown(cSBBadS)
		if recSBBadS.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", recSBBadS.Code)
		}

		// ExportCSV bad project ID
		recCSVBad := httptest.NewRecorder()
		cCSVBad := e.NewContext(reqList, recCSVBad)
		cCSVBad.SetParamNames("projectId")
		cCSVBad.SetParamValues("bad")
		_ = h.ExportCSV(cCSVBad)
		if recCSVBad.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", recCSVBad.Code)
		}

		// ExportSummary bad project ID
		recSumBad := httptest.NewRecorder()
		cSumBad := e.NewContext(reqList, recSumBad)
		cSumBad.SetParamNames("projectId")
		cSumBad.SetParamValues("bad")
		_ = h.ExportSummary(cSumBad)
		if recSumBad.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", recSumBad.Code)
		}

		// ExportCSV with various query params
		recCSVWithParams := httptest.NewRecorder()
		reqCSVWithParams := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/?in_backlog=true&status=TODO&assignee_id=%s", validID.String()), nil)
		cCSVWithParams := e.NewContext(reqCSVWithParams, recCSVWithParams)
		cCSVWithParams.SetParamNames("projectId")
		cCSVWithParams.SetParamValues(validID.String())
		_ = h.ExportCSV(cCSVWithParams)
		if recCSVWithParams.Code != http.StatusOK {
			t.Errorf("expected 200 for CSV with query params, got %d", recCSVWithParams.Code)
		}

		// ExportCSV with sprint_id
		recCSVSprint := httptest.NewRecorder()
		reqCSVSprint := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/?sprint_id=%s", targetID.String()), nil)
		cCSVSprint := e.NewContext(reqCSVSprint, recCSVSprint)
		cCSVSprint.SetParamNames("projectId")
		cCSVSprint.SetParamValues(validID.String())
		_ = h.ExportCSV(cCSVSprint)
		if recCSVSprint.Code != http.StatusOK {
			t.Errorf("expected 200 for CSV with sprint_id, got %d", recCSVSprint.Code)
		}
	})

	// 5. AuthHandler Exhaustive Additional
	t.Run("AuthHandler Exhaustive Additional", func(t *testing.T) {
		h := NewAuthHandler(&mockAuthUsecaseForHandler{})

		// RefreshToken bad json
		recRefBad := httptest.NewRecorder()
		cRefBad := e.NewContext(httptest.NewRequest(http.MethodPost, "/", strings.NewReader("{invalid-json")), recRefBad)
		cRefBad.Request().Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		_ = h.RefreshToken(cRefBad)
		if recRefBad.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for bad json in refresh, got %d", recRefBad.Code)
		}

		// RefreshToken validation error
		recRefVal := httptest.NewRecorder()
		cRefVal := e.NewContext(httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"refresh_token":""}`)), recRefVal)
		cRefVal.Request().Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		_ = h.RefreshToken(cRefVal)
		if recRefVal.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for empty token in refresh, got %d", recRefVal.Code)
		}

		// GoogleLogin bad json
		recGBad := httptest.NewRecorder()
		cGBad := e.NewContext(httptest.NewRequest(http.MethodPost, "/", strings.NewReader("{invalid-json")), recGBad)
		cGBad.Request().Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		_ = h.GoogleLogin(cGBad)
		if recGBad.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for bad json in google login, got %d", recGBad.Code)
		}

		// GoogleLogin validation error
		recGVal := httptest.NewRecorder()
		cGVal := e.NewContext(httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"id_token":""}`)), recGVal)
		cGVal.Request().Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		_ = h.GoogleLogin(cGVal)
		if recGVal.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for empty id_token in google login, got %d", recGVal.Code)
		}

		// GetMe without context user ID -> 401
		recMeUnauth := httptest.NewRecorder()
		cMeUnauth := e.NewContext(httptest.NewRequest(http.MethodGet, "/", nil), recMeUnauth)
		_ = h.GetMe(cMeUnauth)
		if recMeUnauth.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 for GetMe without user context, got %d", recMeUnauth.Code)
		}
	})

	// 6. ProjectHandler & Member Exhaustive Additional
	t.Run("ProjectHandler Exhaustive Additional", func(t *testing.T) {
		hForb := NewProjectHandler(&fullMockProjectUC{err: domain.ErrForbidden})
		hConf := NewProjectHandler(&fullMockProjectUC{err: domain.ErrConflict})
		hNF := NewProjectHandler(&fullMockProjectUC{err: domain.ErrNotFound})
		hErr := NewProjectHandler(&fullMockProjectUC{err: errors.New("server err")})

		// Update without userID
		recUpdUnauth := httptest.NewRecorder()
		cUpdUnauth := e.NewContext(httptest.NewRequest(http.MethodPut, "/", nil), recUpdUnauth)
		cUpdUnauth.SetParamNames("projectId")
		cUpdUnauth.SetParamValues(validID.String())
		_ = hForb.Update(cUpdUnauth)
		if recUpdUnauth.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", recUpdUnauth.Code)
		}

		// Update bad project ID
		recUpdBadP := httptest.NewRecorder()
		cUpdBadP := e.NewContext(httptest.NewRequest(http.MethodPut, "/", nil), recUpdBadP)
		cUpdBadP.Set(middleware.CtxKeyUserID, validID)
		cUpdBadP.SetParamNames("projectId")
		cUpdBadP.SetParamValues("bad")
		_ = hForb.Update(cUpdBadP)
		if recUpdBadP.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", recUpdBadP.Code)
		}

		// Update bad payload
		recUpdBadJSON := httptest.NewRecorder()
		cUpdBadJSON := e.NewContext(httptest.NewRequest(http.MethodPut, "/", strings.NewReader("{bad")), recUpdBadJSON)
		cUpdBadJSON.Request().Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		cUpdBadJSON.Set(middleware.CtxKeyUserID, validID)
		cUpdBadJSON.SetParamNames("projectId")
		cUpdBadJSON.SetParamValues(validID.String())
		_ = hForb.Update(cUpdBadJSON)
		if recUpdBadJSON.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for bad payload in update, got %d", recUpdBadJSON.Code)
		}

		// Delete without userID -> 401
		recDelUnauth := httptest.NewRecorder()
		cDelUnauth := e.NewContext(httptest.NewRequest(http.MethodDelete, "/", nil), recDelUnauth)
		cDelUnauth.SetParamNames("projectId")
		cDelUnauth.SetParamValues(validID.String())
		_ = hForb.Delete(cDelUnauth)
		if recDelUnauth.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", recDelUnauth.Code)
		}

		// Delete bad project ID -> 400
		recDelBadP := httptest.NewRecorder()
		cDelBadP := e.NewContext(httptest.NewRequest(http.MethodDelete, "/", nil), recDelBadP)
		cDelBadP.Set(middleware.CtxKeyUserID, validID)
		cDelBadP.SetParamNames("projectId")
		cDelBadP.SetParamValues("bad")
		_ = hForb.Delete(cDelBadP)
		if recDelBadP.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", recDelBadP.Code)
		}

		// Delete forbidden (not owner) -> 403
		recDelForb := httptest.NewRecorder()
		cDelForb := e.NewContext(httptest.NewRequest(http.MethodDelete, "/", nil), recDelForb)
		cDelForb.Set(middleware.CtxKeyUserID, validID)
		cDelForb.SetParamNames("projectId")
		cDelForb.SetParamValues(validID.String())
		_ = hForb.Delete(cDelForb)
		if recDelForb.Code != http.StatusForbidden {
			t.Errorf("expected 403, got %d", recDelForb.Code)
		}

		// AddMember bad project ID, bad payload, conflict
		recAMBadP := httptest.NewRecorder()
		cAMBadP := e.NewContext(httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"email":"a@b.com","role_name":"ADMIN"}`)), recAMBadP)
		cAMBadP.Request().Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		cAMBadP.SetParamNames("projectId")
		cAMBadP.SetParamValues("bad")
		_ = hConf.AddMember(cAMBadP)
		if recAMBadP.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", recAMBadP.Code)
		}

		recAMBadJSON := httptest.NewRecorder()
		cAMBadJSON := e.NewContext(httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{bad`)), recAMBadJSON)
		cAMBadJSON.Request().Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		cAMBadJSON.SetParamNames("projectId")
		cAMBadJSON.SetParamValues(validID.String())
		_ = hConf.AddMember(cAMBadJSON)
		if recAMBadJSON.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", recAMBadJSON.Code)
		}

		recAMConf := httptest.NewRecorder()
		cAMConf := e.NewContext(httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"email":"a@b.com","role_name":"ADMIN"}`)), recAMConf)
		cAMConf.Request().Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		cAMConf.SetParamNames("projectId")
		cAMConf.SetParamValues(validID.String())
		_ = hConf.AddMember(cAMConf)
		if recAMConf.Code != http.StatusConflict {
			t.Errorf("expected 409 conflict, got %d", recAMConf.Code)
		}

		// UpdateMemberRole bad project ID, bad target user ID, bad json, not found, internal err
		recUMBadP := httptest.NewRecorder()
		cUMBadP := e.NewContext(httptest.NewRequest(http.MethodPut, "/", strings.NewReader(`{"role_name":"MEMBER"}`)), recUMBadP)
		cUMBadP.Request().Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		cUMBadP.SetParamNames("projectId", "userId")
		cUMBadP.SetParamValues("bad", targetID.String())
		_ = hNF.UpdateMemberRole(cUMBadP)
		if recUMBadP.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for bad project ID, got %d", recUMBadP.Code)
		}

		recUMBadU := httptest.NewRecorder()
		cUMBadU := e.NewContext(httptest.NewRequest(http.MethodPut, "/", strings.NewReader(`{"role_name":"MEMBER"}`)), recUMBadU)
		cUMBadU.Request().Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		cUMBadU.SetParamNames("projectId", "userId")
		cUMBadU.SetParamValues(validID.String(), "bad")
		_ = hNF.UpdateMemberRole(cUMBadU)
		if recUMBadU.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for bad user ID, got %d", recUMBadU.Code)
		}

		recUMBadJSON := httptest.NewRecorder()
		cUMBadJSON := e.NewContext(httptest.NewRequest(http.MethodPut, "/", strings.NewReader(`{bad`)), recUMBadJSON)
		cUMBadJSON.Request().Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		cUMBadJSON.SetParamNames("projectId", "userId")
		cUMBadJSON.SetParamValues(validID.String(), targetID.String())
		_ = hNF.UpdateMemberRole(cUMBadJSON)
		if recUMBadJSON.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for bad JSON, got %d", recUMBadJSON.Code)
		}

		recUMNF := httptest.NewRecorder()
		cUMNF := e.NewContext(httptest.NewRequest(http.MethodPut, "/", strings.NewReader(`{"role_name":"MEMBER"}`)), recUMNF)
		cUMNF.Request().Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		cUMNF.SetParamNames("projectId", "userId")
		cUMNF.SetParamValues(validID.String(), targetID.String())
		_ = hNF.UpdateMemberRole(cUMNF)
		if recUMNF.Code != http.StatusNotFound {
			t.Errorf("expected 404, got %d", recUMNF.Code)
		}

		recUMErr := httptest.NewRecorder()
		cUMErr := e.NewContext(httptest.NewRequest(http.MethodPut, "/", strings.NewReader(`{"role_name":"MEMBER"}`)), recUMErr)
		cUMErr.Request().Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		cUMErr.SetParamNames("projectId", "userId")
		cUMErr.SetParamValues(validID.String(), targetID.String())
		_ = hErr.UpdateMemberRole(cUMErr)
		if recUMErr.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", recUMErr.Code)
		}

		// RemoveMember bad project ID, bad user ID, forbidden
		recRMBadP := httptest.NewRecorder()
		cRMBadP := e.NewContext(httptest.NewRequest(http.MethodDelete, "/", nil), recRMBadP)
		cRMBadP.SetParamNames("projectId", "userId")
		cRMBadP.SetParamValues("bad", targetID.String())
		_ = hForb.RemoveMember(cRMBadP)
		if recRMBadP.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for bad project ID, got %d", recRMBadP.Code)
		}

		recRMBadU := httptest.NewRecorder()
		cRMBadU := e.NewContext(httptest.NewRequest(http.MethodDelete, "/", nil), recRMBadU)
		cRMBadU.SetParamNames("projectId", "userId")
		cRMBadU.SetParamValues(validID.String(), "bad")
		_ = hForb.RemoveMember(cRMBadU)
		if recRMBadU.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for bad user ID, got %d", recRMBadU.Code)
		}

		recRMForb := httptest.NewRecorder()
		cRMForb := e.NewContext(httptest.NewRequest(http.MethodDelete, "/", nil), recRMForb)
		cRMForb.SetParamNames("projectId", "userId")
		cRMForb.SetParamValues(validID.String(), targetID.String())
		_ = hForb.RemoveMember(cRMForb)
		if recRMForb.Code != http.StatusForbidden {
			t.Errorf("expected 403 for removing owner, got %d", recRMForb.Code)
		}
	})

	// 7. IssueHandler List Query Params Exhaustive
	t.Run("IssueHandler List Query Params", func(t *testing.T) {
		h := NewIssueHandler(&mockIssueUsecaseForHandler{})

		// Query with in_backlog, type, status, parent_id, assignee_id, page, limit, search, sort
		queryURL := fmt.Sprintf("/?in_backlog=true&type=TASK&status=TODO&parent_id=%s&assignee_id=%s&page=2&limit=25&search=test&sort_by=created_at&sort_order=asc", validID.String(), targetID.String())
		rec := httptest.NewRecorder()
		c := e.NewContext(httptest.NewRequest(http.MethodGet, queryURL, nil), rec)
		c.SetParamNames("projectId")
		c.SetParamValues(validID.String())
		_ = h.List(c)
		if rec.Code != http.StatusOK {
			t.Errorf("expected 200 for filtered list, got %d", rec.Code)
		}

		// Query with sprint_id
		querySprintURL := fmt.Sprintf("/?sprint_id=%s", targetID.String())
		recSprint := httptest.NewRecorder()
		cSprint := e.NewContext(httptest.NewRequest(http.MethodGet, querySprintURL, nil), recSprint)
		cSprint.SetParamNames("projectId")
		cSprint.SetParamValues(validID.String())
		_ = h.List(cSprint)
		if recSprint.Code != http.StatusOK {
			t.Errorf("expected 200 for sprint_id list, got %d", recSprint.Code)
		}
	})
}
