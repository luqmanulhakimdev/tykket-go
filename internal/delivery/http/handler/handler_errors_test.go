package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/luqmanulhakimdev/tykket-go/internal/delivery/http/middleware"
	"github.com/luqmanulhakimdev/tykket-go/internal/domain"
)

type failingAuthUsecase struct {
	domain.AuthUsecase
}

func (f *failingAuthUsecase) Register(ctx context.Context, req domain.RegisterRequest) (*domain.AuthResponse, error) {
	return nil, errors.New("db error")
}

func (f *failingAuthUsecase) Login(ctx context.Context, req domain.LoginRequest) (*domain.AuthResponse, error) {
	return nil, errors.New("server error")
}

func (f *failingAuthUsecase) GetMe(ctx context.Context, userID uuid.UUID) (*domain.User, error) {
	return nil, domain.ErrNotFound
}

type failingProjectUsecase struct {
	domain.ProjectUsecase
}

func (f *failingProjectUsecase) CreateProject(ctx context.Context, userID uuid.UUID, req domain.CreateProjectRequest) (*domain.Project, error) {
	if req.Key == "CONFLICT" {
		return nil, domain.ErrConflict
	}
	if req.Key == "INVALID" {
		return nil, domain.ErrInvalidInput
	}
	return nil, errors.New("db fail")
}

func (f *failingProjectUsecase) ListProjects(ctx context.Context, userID uuid.UUID) ([]domain.Project, error) {
	return nil, errors.New("db fail")
}

func (f *failingProjectUsecase) GetProjectByID(ctx context.Context, userID, projectID uuid.UUID) (*domain.Project, error) {
	if projectID == uuid.Nil {
		return nil, domain.ErrNotFound
	}
	return nil, errors.New("db fail")
}

func (f *failingProjectUsecase) UpdateProject(ctx context.Context, userID, projectID uuid.UUID, req domain.UpdateProjectRequest) (*domain.Project, error) {
	if req.Name == "NOTFOUND" {
		return nil, domain.ErrNotFound
	}
	return nil, errors.New("db fail")
}

func (f *failingProjectUsecase) DeleteProject(ctx context.Context, userID, projectID uuid.UUID) error {
	if projectID == uuid.Nil {
		return domain.ErrNotFound
	}
	if userID == uuid.Nil {
		return domain.ErrForbidden
	}
	return errors.New("db fail")
}

func (f *failingProjectUsecase) ListMembers(ctx context.Context, projectID uuid.UUID) ([]domain.ProjectMember, error) {
	return nil, errors.New("db fail")
}

func (f *failingProjectUsecase) AddMember(ctx context.Context, projectID uuid.UUID, req domain.AddMemberRequest) (*domain.ProjectMember, error) {
	if req.Email == "conflict@example.com" {
		return nil, domain.ErrConflict
	}
	if req.Email == "notfound@example.com" {
		return nil, domain.ErrNotFound
	}
	return nil, errors.New("db fail")
}

func (f *failingProjectUsecase) UpdateMemberRole(ctx context.Context, projectID, targetUserID uuid.UUID, req domain.UpdateMemberRoleRequest) (*domain.ProjectMember, error) {
	if targetUserID == uuid.Nil {
		return nil, domain.ErrNotFound
	}
	return nil, errors.New("db fail")
}

func (f *failingProjectUsecase) RemoveMember(ctx context.Context, projectID, targetUserID uuid.UUID) error {
	if targetUserID == uuid.Nil {
		return domain.ErrNotFound
	}
	if projectID == uuid.Nil {
		return domain.ErrForbidden
	}
	return errors.New("db fail")
}

type failingSprintUsecase struct {
	domain.SprintUsecase
}

func (f *failingSprintUsecase) CreateSprint(ctx context.Context, projectID uuid.UUID, req domain.CreateSprintRequest) (*domain.Sprint, error) {
	return nil, errors.New("fail")
}

func (f *failingSprintUsecase) ListSprints(ctx context.Context, projectID uuid.UUID) ([]domain.Sprint, error) {
	return nil, errors.New("fail")
}

func (f *failingSprintUsecase) GetSprintByID(ctx context.Context, projectID, sprintID uuid.UUID) (*domain.Sprint, error) {
	if sprintID == uuid.Nil {
		return nil, domain.ErrNotFound
	}
	return nil, errors.New("fail")
}

func (f *failingSprintUsecase) UpdateSprint(ctx context.Context, projectID, sprintID uuid.UUID, req domain.UpdateSprintRequest) (*domain.Sprint, error) {
	if sprintID == uuid.Nil {
		return nil, domain.ErrNotFound
	}
	return nil, errors.New("fail")
}

func (f *failingSprintUsecase) StartSprint(ctx context.Context, projectID, sprintID uuid.UUID, req domain.StartSprintRequest) (*domain.Sprint, error) {
	if sprintID == uuid.Nil {
		return nil, domain.ErrActiveSprintExists
	}
	if projectID == uuid.Nil {
		return nil, domain.ErrInvalidInput
	}
	return nil, errors.New("fail")
}

func (f *failingSprintUsecase) CompleteSprint(ctx context.Context, projectID, sprintID uuid.UUID, req domain.CompleteSprintRequest) (*domain.Sprint, error) {
	if sprintID == uuid.Nil {
		return nil, domain.ErrInvalidInput
	}
	return nil, errors.New("fail")
}

func (f *failingSprintUsecase) DeleteSprint(ctx context.Context, projectID, sprintID uuid.UUID) error {
	if sprintID == uuid.Nil {
		return domain.ErrNotFound
	}
	return errors.New("fail")
}

func (f *failingSprintUsecase) BulkMoveIssues(ctx context.Context, projectID uuid.UUID, req domain.BulkMoveSprintIssuesRequest) error {
	return errors.New("fail")
}

type failingIssueUsecase struct {
	domain.IssueUsecase
}

func (f *failingIssueUsecase) CreateIssue(ctx context.Context, reporterID, projectID uuid.UUID, req domain.CreateIssueRequest) (*domain.Issue, error) {
	if req.Title == "HIERARCHY" {
		return nil, domain.ErrInvalidHierarchy
	}
	if req.Title == "NOTFOUND" {
		return nil, domain.ErrNotFound
	}
	return nil, errors.New("fail")
}

func (f *failingIssueUsecase) ListIssues(ctx context.Context, filter domain.IssueFilter) ([]domain.Issue, int64, error) {
	return nil, 0, errors.New("fail")
}

func (f *failingIssueUsecase) GetIssueByID(ctx context.Context, projectID, issueID uuid.UUID) (*domain.Issue, error) {
	if issueID == uuid.Nil {
		return nil, domain.ErrNotFound
	}
	return nil, errors.New("fail")
}

func (f *failingIssueUsecase) GetIssueByKey(ctx context.Context, key string) (*domain.Issue, error) {
	if key == "MISSING" {
		return nil, domain.ErrNotFound
	}
	return nil, errors.New("fail")
}

func (f *failingIssueUsecase) UpdateIssue(ctx context.Context, projectID, issueID uuid.UUID, req domain.UpdateIssueRequest) (*domain.Issue, error) {
	if issueID == uuid.Nil {
		return nil, domain.ErrNotFound
	}
	if req.Title != nil && *req.Title == "HIERARCHY" {
		return nil, domain.ErrInvalidHierarchy
	}
	return nil, errors.New("fail")
}

func (f *failingIssueUsecase) MoveIssue(ctx context.Context, projectID, issueID uuid.UUID, req domain.MoveIssueRequest) (*domain.Issue, error) {
	if issueID == uuid.Nil {
		return nil, domain.ErrNotFound
	}
	return nil, errors.New("fail")
}

func (f *failingIssueUsecase) DeleteIssue(ctx context.Context, projectID, issueID uuid.UUID) error {
	if issueID == uuid.Nil {
		return domain.ErrNotFound
	}
	return errors.New("fail")
}

type failingMetricUsecase struct {
	domain.MetricUsecase
}

func (f *failingMetricUsecase) GetProjectMetrics(ctx context.Context, projectID uuid.UUID) (*domain.ProjectSummaryMetrics, error) {
	return nil, errors.New("fail")
}

func (f *failingMetricUsecase) GetSprintMetrics(ctx context.Context, projectID, sprintID uuid.UUID) (*domain.SprintMetrics, error) {
	return nil, errors.New("fail")
}

func (f *failingMetricUsecase) ExportProjectSummary(ctx context.Context, projectID uuid.UUID) (interface{}, error) {
	return nil, errors.New("fail")
}

func TestHandler_AllErrorBranches(t *testing.T) {
	e := setupEcho()
	authH := NewAuthHandler(&failingAuthUsecase{})
	projH := NewProjectHandler(&failingProjectUsecase{})
	sprintH := NewSprintHandler(&failingSprintUsecase{})
	issueH := NewIssueHandler(&failingIssueUsecase{})
	metricH := NewMetricHandler(&failingMetricUsecase{})

	validID := uuid.New()

	t.Run("AuthHandler server error paths", func(t *testing.T) {
		body, _ := json.Marshal(domain.RegisterRequest{
			Name:     "Test",
			Email:    "test@example.com",
			Password: "password123",
		})
		req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		_ = authH.Register(e.NewContext(req, rec))
		if rec.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", rec.Code)
		}

		bodyLogin, _ := json.Marshal(domain.LoginRequest{
			Email:    "test@example.com",
			Password: "password123",
		})
		reqLogin := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(bodyLogin))
		reqLogin.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		recLogin := httptest.NewRecorder()
		_ = authH.Login(e.NewContext(reqLogin, recLogin))
		if recLogin.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", recLogin.Code)
		}

		reqMe := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
		recMe := httptest.NewRecorder()
		cMe := e.NewContext(reqMe, recMe)
		cMe.Set(middleware.CtxKeyUserID, uuid.New())
		_ = authH.GetMe(cMe)
		if recMe.Code != http.StatusNotFound {
			t.Errorf("expected 404, got %d", recMe.Code)
		}
	})

	t.Run("ProjectHandler error paths", func(t *testing.T) {
		reqList := httptest.NewRequest(http.MethodGet, "/", nil)

		// Create unauthorized
		recUnauth := httptest.NewRecorder()
		_ = projH.Create(e.NewContext(reqList, recUnauth))
		if recUnauth.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", recUnauth.Code)
		}

		// Create conflict
		bodyConf, _ := json.Marshal(domain.CreateProjectRequest{Name: "Proj", Key: "CONFLICT"})
		reqConf := httptest.NewRequest(http.MethodPost, "/projects", bytes.NewReader(bodyConf))
		reqConf.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		recConf := httptest.NewRecorder()
		cConf := e.NewContext(reqConf, recConf)
		cConf.Set(middleware.CtxKeyUserID, validID)
		_ = projH.Create(cConf)
		if recConf.Code != http.StatusConflict {
			t.Errorf("expected 409, got %d", recConf.Code)
		}

		// Create invalid
		bodyInv, _ := json.Marshal(domain.CreateProjectRequest{Name: "Proj", Key: "INVALID"})
		reqInv := httptest.NewRequest(http.MethodPost, "/projects", bytes.NewReader(bodyInv))
		reqInv.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		recInv := httptest.NewRecorder()
		cInv := e.NewContext(reqInv, recInv)
		cInv.Set(middleware.CtxKeyUserID, validID)
		_ = projH.Create(cInv)
		if recInv.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", recInv.Code)
		}

		// Create server error
		bodyErr, _ := json.Marshal(domain.CreateProjectRequest{Name: "Proj", Key: "FAILERR"})
		reqErr := httptest.NewRequest(http.MethodPost, "/projects", bytes.NewReader(bodyErr))
		reqErr.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		recErr := httptest.NewRecorder()
		cErr := e.NewContext(reqErr, recErr)
		cErr.Set(middleware.CtxKeyUserID, validID)
		_ = projH.Create(cErr)
		if recErr.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", recErr.Code)
		}

		// List error
		recList := httptest.NewRecorder()
		cList := e.NewContext(reqList, recList)
		cList.Set(middleware.CtxKeyUserID, validID)
		_ = projH.List(cList)
		if recList.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", recList.Code)
		}

		// GetByID not found and server error
		recGetNF := httptest.NewRecorder()
		cGetNF := e.NewContext(reqList, recGetNF)
		cGetNF.Set(middleware.CtxKeyUserID, validID)
		cGetNF.SetParamNames("projectId")
		cGetNF.SetParamValues(uuid.Nil.String())
		_ = projH.GetByID(cGetNF)
		if recGetNF.Code != http.StatusNotFound {
			t.Errorf("expected 404, got %d", recGetNF.Code)
		}

		recGetErr := httptest.NewRecorder()
		cGetErr := e.NewContext(reqList, recGetErr)
		cGetErr.Set(middleware.CtxKeyUserID, validID)
		cGetErr.SetParamNames("projectId")
		cGetErr.SetParamValues(validID.String())
		_ = projH.GetByID(cGetErr)
		if recGetErr.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", recGetErr.Code)
		}

		// Update invalid ID, not found, server error
		recUpdBadID := httptest.NewRecorder()
		cUpdBadID := e.NewContext(reqList, recUpdBadID)
		cUpdBadID.Set(middleware.CtxKeyUserID, validID)
		cUpdBadID.SetParamNames("projectId")
		cUpdBadID.SetParamValues("bad")
		_ = projH.Update(cUpdBadID)
		if recUpdBadID.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for bad id, got %d", recUpdBadID.Code)
		}

		recUpdNF := httptest.NewRecorder()
		reqUpdNF := httptest.NewRequest(http.MethodPut, "/", bytes.NewReader([]byte(`{"name":"NOTFOUND"}`)))
		reqUpdNF.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		cUpdNF := e.NewContext(reqUpdNF, recUpdNF)
		cUpdNF.Set(middleware.CtxKeyUserID, validID)
		cUpdNF.SetParamNames("projectId")
		cUpdNF.SetParamValues(validID.String())
		_ = projH.Update(cUpdNF)
		if recUpdNF.Code != http.StatusNotFound {
			t.Errorf("expected 404, got %d", recUpdNF.Code)
		}

		// Delete errors
		recDelBad := httptest.NewRecorder()
		cDelBad := e.NewContext(reqList, recDelBad)
		cDelBad.Set(middleware.CtxKeyUserID, validID)
		cDelBad.SetParamNames("projectId")
		cDelBad.SetParamValues("bad")
		_ = projH.Delete(cDelBad)
		if recDelBad.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", recDelBad.Code)
		}

		recDelF := httptest.NewRecorder()
		cDelF := e.NewContext(reqList, recDelF)
		cDelF.Set(middleware.CtxKeyUserID, uuid.Nil)
		cDelF.SetParamNames("projectId")
		cDelF.SetParamValues(validID.String())
		_ = projH.Delete(cDelF)
		if recDelF.Code != http.StatusForbidden {
			t.Errorf("expected 403, got %d", recDelF.Code)
		}

		recDelNF := httptest.NewRecorder()
		cDelNF := e.NewContext(reqList, recDelNF)
		cDelNF.Set(middleware.CtxKeyUserID, validID)
		cDelNF.SetParamNames("projectId")
		cDelNF.SetParamValues(uuid.Nil.String())
		_ = projH.Delete(cDelNF)
		if recDelNF.Code != http.StatusNotFound {
			t.Errorf("expected 404, got %d", recDelNF.Code)
		}

		// Member operations errors
		recMemAddBad := httptest.NewRecorder()
		cMemAddBad := e.NewContext(reqList, recMemAddBad)
		cMemAddBad.SetParamNames("projectId")
		cMemAddBad.SetParamValues("bad")
		_ = projH.AddMember(cMemAddBad)
		if recMemAddBad.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", recMemAddBad.Code)
		}

		recMemAddConf := httptest.NewRecorder()
		reqMemAddConf := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte(`{"email":"conflict@example.com","role_name":"MEMBER"}`)))
		reqMemAddConf.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		cMemAddConf := e.NewContext(reqMemAddConf, recMemAddConf)
		cMemAddConf.SetParamNames("projectId")
		cMemAddConf.SetParamValues(validID.String())
		_ = projH.AddMember(cMemAddConf)
		if recMemAddConf.Code != http.StatusConflict {
			t.Errorf("expected 409, got %d", recMemAddConf.Code)
		}

		recMemAddNF := httptest.NewRecorder()
		reqMemAddNF := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte(`{"email":"notfound@example.com","role_name":"MEMBER"}`)))
		reqMemAddNF.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		cMemAddNF := e.NewContext(reqMemAddNF, recMemAddNF)
		cMemAddNF.SetParamNames("projectId")
		cMemAddNF.SetParamValues(validID.String())
		_ = projH.AddMember(cMemAddNF)
		if recMemAddNF.Code != http.StatusNotFound {
			t.Errorf("expected 404, got %d", recMemAddNF.Code)
		}

		// Member role bad user id
		recBadUID := httptest.NewRecorder()
		cBadUID := e.NewContext(reqList, recBadUID)
		cBadUID.SetParamNames("projectId", "userId")
		cBadUID.SetParamValues(validID.String(), "bad-uuid")
		_ = projH.UpdateMemberRole(cBadUID)
		if recBadUID.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for bad user id, got %d", recBadUID.Code)
		}

		// Remove member bad user id
		recRemBadUID := httptest.NewRecorder()
		cRemBadUID := e.NewContext(reqList, recRemBadUID)
		cRemBadUID.SetParamNames("projectId", "userId")
		cRemBadUID.SetParamValues(validID.String(), "bad-uuid")
		_ = projH.RemoveMember(cRemBadUID)
		if recRemBadUID.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for bad user id, got %d", recRemBadUID.Code)
		}

		recMemRemF := httptest.NewRecorder()
		cMemRemF := e.NewContext(reqList, recMemRemF)
		cMemRemF.SetParamNames("projectId", "userId")
		cMemRemF.SetParamValues(uuid.Nil.String(), validID.String())
		_ = projH.RemoveMember(cMemRemF)
		if recMemRemF.Code != http.StatusForbidden {
			t.Errorf("expected 403, got %d", recMemRemF.Code)
		}
	})

	t.Run("SprintHandler error paths", func(t *testing.T) {
		reqList := httptest.NewRequest(http.MethodGet, "/", nil)

		// Create with bad project ID
		recCrBad := httptest.NewRecorder()
		cCrBad := e.NewContext(reqList, recCrBad)
		cCrBad.SetParamNames("projectId")
		cCrBad.SetParamValues("bad")
		_ = sprintH.Create(cCrBad)
		if recCrBad.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", recCrBad.Code)
		}

		// Create server error
		recCrErr := httptest.NewRecorder()
		reqCrErr := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte(`{"name":"Sprint"}`)))
		reqCrErr.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		cCrErr := e.NewContext(reqCrErr, recCrErr)
		cCrErr.SetParamNames("projectId")
		cCrErr.SetParamValues(validID.String())
		_ = sprintH.Create(cCrErr)
		if recCrErr.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", recCrErr.Code)
		}

		// List error
		recList := httptest.NewRecorder()
		cList := e.NewContext(reqList, recList)
		cList.SetParamNames("projectId")
		cList.SetParamValues(validID.String())
		_ = sprintH.List(cList)
		if recList.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", recList.Code)
		}

		// Start conflict and bad input
		recStartConf := httptest.NewRecorder()
		reqStart := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte(`{}`)))
		reqStart.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		cStartConf := e.NewContext(reqStart, recStartConf)
		cStartConf.SetParamNames("projectId", "sprintId")
		cStartConf.SetParamValues(validID.String(), uuid.Nil.String())
		_ = sprintH.Start(cStartConf)
		if recStartConf.Code != http.StatusConflict {
			t.Errorf("expected 409, got %d", recStartConf.Code)
		}

		recStartInv := httptest.NewRecorder()
		cStartInv := e.NewContext(reqStart, recStartInv)
		cStartInv.SetParamNames("projectId", "sprintId")
		cStartInv.SetParamValues(uuid.Nil.String(), validID.String())
		_ = sprintH.Start(cStartInv)
		if recStartInv.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", recStartInv.Code)
		}

		// Complete bad input
		recCompInv := httptest.NewRecorder()
		cCompInv := e.NewContext(reqStart, recCompInv)
		cCompInv.SetParamNames("projectId", "sprintId")
		cCompInv.SetParamValues(validID.String(), uuid.Nil.String())
		_ = sprintH.Complete(cCompInv)
		if recCompInv.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", recCompInv.Code)
		}

		// Sprint bad sprint id on Get, Update, Delete
		recSprBadSID := httptest.NewRecorder()
		cSprBadSID := e.NewContext(reqList, recSprBadSID)
		cSprBadSID.SetParamNames("projectId", "sprintId")
		cSprBadSID.SetParamValues(validID.String(), "bad-uuid")
		_ = sprintH.GetByID(cSprBadSID)
		if recSprBadSID.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for bad sprint id, got %d", recSprBadSID.Code)
		}

		recUpdBadSID := httptest.NewRecorder()
		cUpdBadSID := e.NewContext(reqList, recUpdBadSID)
		cUpdBadSID.SetParamNames("projectId", "sprintId")
		cUpdBadSID.SetParamValues(validID.String(), "bad-uuid")
		_ = sprintH.Update(cUpdBadSID)
		if recUpdBadSID.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for bad sprint id, got %d", recUpdBadSID.Code)
		}

		recDelBadSID := httptest.NewRecorder()
		cDelBadSID := e.NewContext(reqList, recDelBadSID)
		cDelBadSID.SetParamNames("projectId", "sprintId")
		cDelBadSID.SetParamValues(validID.String(), "bad-uuid")
		_ = sprintH.Delete(cDelBadSID)
		if recDelBadSID.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for bad sprint id, got %d", recDelBadSID.Code)
		}

		// BulkMove bad project id
		recBMBadPID := httptest.NewRecorder()
		cBMBadPID := e.NewContext(reqList, recBMBadPID)
		cBMBadPID.SetParamNames("projectId")
		cBMBadPID.SetParamValues("bad-uuid")
		_ = sprintH.BulkMove(cBMBadPID)
		if recBMBadPID.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for bad project id on bulk move, got %d", recBMBadPID.Code)
		}

		// Delete not found
		recDelNF := httptest.NewRecorder()
		cDelNF := e.NewContext(reqList, recDelNF)
		cDelNF.SetParamNames("projectId", "sprintId")
		cDelNF.SetParamValues(validID.String(), uuid.Nil.String())
		_ = sprintH.Delete(cDelNF)
		if recDelNF.Code != http.StatusNotFound {
			t.Errorf("expected 404, got %d", recDelNF.Code)
		}
	})

	t.Run("IssueHandler error paths", func(t *testing.T) {
		reqList := httptest.NewRequest(http.MethodGet, "/", nil)

		// Create hierarchy error
		bodyHier, _ := json.Marshal(domain.CreateIssueRequest{
			Type:     domain.IssueTypeEpic,
			Priority: domain.IssuePriorityHigh,
			Title:    "HIERARCHY",
		})
		reqHier := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(bodyHier))
		reqHier.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		recHier := httptest.NewRecorder()
		cHier := e.NewContext(reqHier, recHier)
		cHier.Set(middleware.CtxKeyUserID, validID)
		cHier.SetParamNames("projectId")
		cHier.SetParamValues(validID.String())
		_ = issueH.Create(cHier)
		if recHier.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", recHier.Code)
		}

		// Create not found error
		bodyNF, _ := json.Marshal(domain.CreateIssueRequest{
			Type:     domain.IssueTypeTask,
			Priority: domain.IssuePriorityHigh,
			Title:    "NOTFOUND",
		})
		reqNF := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(bodyNF))
		reqNF.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		recNF := httptest.NewRecorder()
		cNF := e.NewContext(reqNF, recNF)
		cNF.Set(middleware.CtxKeyUserID, validID)
		cNF.SetParamNames("projectId")
		cNF.SetParamValues(validID.String())
		_ = issueH.Create(cNF)
		if recNF.Code != http.StatusNotFound {
			t.Errorf("expected 404, got %d", recNF.Code)
		}

		// List error
		recList := httptest.NewRecorder()
		cList := e.NewContext(reqList, recList)
		cList.SetParamNames("projectId")
		cList.SetParamValues(validID.String())
		_ = issueH.List(cList)
		if recList.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", recList.Code)
		}

		// GetByID not found
		recGetNF := httptest.NewRecorder()
		cGetNF := e.NewContext(reqList, recGetNF)
		cGetNF.SetParamNames("projectId", "issueId")
		cGetNF.SetParamValues(validID.String(), uuid.Nil.String())
		_ = issueH.GetByID(cGetNF)
		if recGetNF.Code != http.StatusNotFound {
			t.Errorf("expected 404, got %d", recGetNF.Code)
		}

		// GetByKey missing
		recKeyNF := httptest.NewRecorder()
		cKeyNF := e.NewContext(reqList, recKeyNF)
		cKeyNF.SetParamNames("key")
		cKeyNF.SetParamValues("MISSING")
		_ = issueH.GetByKey(cKeyNF)
		if recKeyNF.Code != http.StatusNotFound {
			t.Errorf("expected 404, got %d", recKeyNF.Code)
		}

		// Update not found
		title := "Updated"
		bodyUpd, _ := json.Marshal(domain.UpdateIssueRequest{Title: &title})
		reqUpd := httptest.NewRequest(http.MethodPut, "/", bytes.NewReader(bodyUpd))
		reqUpd.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		recUpdNF := httptest.NewRecorder()
		cUpdNF := e.NewContext(reqUpd, recUpdNF)
		cUpdNF.SetParamNames("projectId", "issueId")
		cUpdNF.SetParamValues(validID.String(), uuid.Nil.String())
		_ = issueH.Update(cUpdNF)
		if recUpdNF.Code != http.StatusNotFound {
			t.Errorf("expected 404, got %d", recUpdNF.Code)
		}

		// Move not found
		bodyMove, _ := json.Marshal(domain.MoveIssueRequest{Position: 100.0})
		reqMove := httptest.NewRequest(http.MethodPatch, "/", bytes.NewReader(bodyMove))
		reqMove.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		recMoveNF := httptest.NewRecorder()
		cMoveNF := e.NewContext(reqMove, recMoveNF)
		cMoveNF.SetParamNames("projectId", "issueId")
		cMoveNF.SetParamValues(validID.String(), uuid.Nil.String())
		_ = issueH.Move(cMoveNF)
		if recMoveNF.Code != http.StatusNotFound {
			t.Errorf("expected 404, got %d", recMoveNF.Code)
		}

		// Sprint bad sprint id on Get, Update, Delete
		recSprBadSID := httptest.NewRecorder()
		cSprBadSID := e.NewContext(reqList, recSprBadSID)
		cSprBadSID.SetParamNames("projectId", "sprintId")
		cSprBadSID.SetParamValues(validID.String(), "bad-uuid")
		_ = sprintH.GetByID(cSprBadSID)
		if recSprBadSID.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for bad sprint id, got %d", recSprBadSID.Code)
		}

		recUpdBadSID := httptest.NewRecorder()
		cUpdBadSID := e.NewContext(reqList, recUpdBadSID)
		cUpdBadSID.SetParamNames("projectId", "sprintId")
		cUpdBadSID.SetParamValues(validID.String(), "bad-uuid")
		_ = sprintH.Update(cUpdBadSID)
		if recUpdBadSID.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for bad sprint id, got %d", recUpdBadSID.Code)
		}

		recDelBadSID := httptest.NewRecorder()
		cDelBadSID := e.NewContext(reqList, recDelBadSID)
		cDelBadSID.SetParamNames("projectId", "sprintId")
		cDelBadSID.SetParamValues(validID.String(), "bad-uuid")
		_ = sprintH.Delete(cDelBadSID)
		if recDelBadSID.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for bad sprint id, got %d", recDelBadSID.Code)
		}

		// BulkMove bad project id
		recBMBadPID := httptest.NewRecorder()
		cBMBadPID := e.NewContext(reqList, recBMBadPID)
		cBMBadPID.SetParamNames("projectId")
		cBMBadPID.SetParamValues("bad-uuid")
		_ = sprintH.BulkMove(cBMBadPID)
		if recBMBadPID.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for bad project id on bulk move, got %d", recBMBadPID.Code)
		}

		// Delete not found
		recDelNF := httptest.NewRecorder()
		cDelNF := e.NewContext(reqList, recDelNF)
		cDelNF.SetParamNames("projectId", "issueId")
		cDelNF.SetParamValues(validID.String(), uuid.Nil.String())
		_ = issueH.Delete(cDelNF)
		if recDelNF.Code != http.StatusNotFound {
			t.Errorf("expected 404, got %d", recDelNF.Code)
		}
	})

	t.Run("MetricHandler error paths", func(t *testing.T) {
		reqList := httptest.NewRequest(http.MethodGet, "/", nil)

		// GetProjectMetrics error
		recPM := httptest.NewRecorder()
		cPM := e.NewContext(reqList, recPM)
		cPM.SetParamNames("projectId")
		cPM.SetParamValues(validID.String())
		_ = metricH.GetProjectMetrics(cPM)
		if recPM.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", recPM.Code)
		}

		// GetSprintBurndown error
		recSB := httptest.NewRecorder()
		cSB := e.NewContext(reqList, recSB)
		cSB.SetParamNames("projectId", "sprintId")
		cSB.SetParamValues(validID.String(), validID.String())
		_ = metricH.GetSprintBurndown(cSB)
		if recSB.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", recSB.Code)
		}

		// ExportSummary error
		recSum := httptest.NewRecorder()
		cSum := e.NewContext(reqList, recSum)
		cSum.SetParamNames("projectId")
		cSum.SetParamValues(validID.String())
		_ = metricH.ExportSummary(cSum)
		if recSum.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", recSum.Code)
		}
	})
}
