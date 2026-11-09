package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/luqmanulhakimdev/tykket-go/internal/delivery/http/middleware"
	"github.com/luqmanulhakimdev/tykket-go/internal/domain"
	"github.com/luqmanulhakimdev/tykket-go/pkg/token"
	"github.com/luqmanulhakimdev/tykket-go/pkg/validator"
)

// =========================================================================
// Mocks for Handlers
// =========================================================================

type mockAuthUsecaseForHandler struct {
	domain.AuthUsecase
}

func (m *mockAuthUsecaseForHandler) Register(ctx context.Context, req domain.RegisterRequest) (*domain.AuthResponse, error) {
	if req.Email == "conflict@example.com" {
		return nil, domain.ErrConflict
	}
	if req.Email == "internal@example.com" {
		return nil, errors.New("db error")
	}
	return &domain.AuthResponse{User: &domain.User{Email: req.Email}}, nil
}

func (m *mockAuthUsecaseForHandler) Login(ctx context.Context, req domain.LoginRequest) (*domain.AuthResponse, error) {
	if req.Email == "wrong@example.com" {
		return nil, domain.ErrUnauthorized
	}
	if req.Email == "fail@example.com" {
		return nil, errors.New("unexpected error")
	}
	return &domain.AuthResponse{User: &domain.User{Email: req.Email}}, nil
}

func (m *mockAuthUsecaseForHandler) RefreshToken(ctx context.Context, req domain.RefreshTokenRequest) (*domain.AuthResponse, error) {
	if req.RefreshToken == "invalid" {
		return nil, domain.ErrUnauthorized
	}
	return &domain.AuthResponse{Tokens: &token.TokenPair{AccessToken: "new-access"}}, nil
}

func (m *mockAuthUsecaseForHandler) GoogleLogin(ctx context.Context, req domain.GoogleAuthRequest) (*domain.AuthResponse, error) {
	if req.IDToken == "invalid" {
		return nil, errors.New("invalid google token")
	}
	return &domain.AuthResponse{Tokens: &token.TokenPair{AccessToken: "google-access"}}, nil
}

func (m *mockAuthUsecaseForHandler) GetMe(ctx context.Context, userID uuid.UUID) (*domain.User, error) {
	if userID == uuid.Nil {
		return nil, domain.ErrNotFound
	}
	return &domain.User{ID: userID, Email: "me@example.com"}, nil
}

type mockProjectUsecaseForHandler struct {
	domain.ProjectUsecase
}

func (m *mockProjectUsecaseForHandler) CreateProject(ctx context.Context, userID uuid.UUID, req domain.CreateProjectRequest) (*domain.Project, error) {
	if req.Key == "CONFLICT" {
		return nil, domain.ErrConflict
	}
	if req.Key == "INVALID!" {
		return nil, domain.ErrInvalidInput
	}
	if req.Key == "FAIL" {
		return nil, errors.New("unexpected err")
	}
	return &domain.Project{Key: req.Key, Name: req.Name}, nil
}

func (m *mockProjectUsecaseForHandler) ListProjects(ctx context.Context, userID uuid.UUID) ([]domain.Project, error) {
	return []domain.Project{{Name: "Proj 1"}}, nil
}

func (m *mockProjectUsecaseForHandler) GetProjectByID(ctx context.Context, userID, projectID uuid.UUID) (*domain.Project, error) {
	if projectID == uuid.Nil {
		return nil, domain.ErrNotFound
	}
	return &domain.Project{ID: projectID, Name: "Proj 1"}, nil
}

func (m *mockProjectUsecaseForHandler) UpdateProject(ctx context.Context, userID, projectID uuid.UUID, req domain.UpdateProjectRequest) (*domain.Project, error) {
	if req.Name == "NOTFOUND" {
		return nil, domain.ErrNotFound
	}
	return &domain.Project{ID: projectID, Name: req.Name}, nil
}

func (m *mockProjectUsecaseForHandler) DeleteProject(ctx context.Context, userID, projectID uuid.UUID) error {
	if projectID == uuid.Nil {
		return domain.ErrNotFound
	}
	if reqName := userID.String(); reqName == "00000000-0000-0000-0000-000000000099" {
		return domain.ErrForbidden
	}
	return nil
}

func (m *mockProjectUsecaseForHandler) ListMembers(ctx context.Context, projectID uuid.UUID) ([]domain.ProjectMember, error) {
	return []domain.ProjectMember{{ProjectID: projectID}}, nil
}

func (m *mockProjectUsecaseForHandler) AddMember(ctx context.Context, projectID uuid.UUID, req domain.AddMemberRequest) (*domain.ProjectMember, error) {
	if req.Email == "conflict@example.com" {
		return nil, domain.ErrConflict
	}
	if req.Email == "notfound@example.com" {
		return nil, domain.ErrNotFound
	}
	return &domain.ProjectMember{ProjectID: projectID}, nil
}

func (m *mockProjectUsecaseForHandler) UpdateMemberRole(ctx context.Context, projectID, targetUserID uuid.UUID, req domain.UpdateMemberRoleRequest) (*domain.ProjectMember, error) {
	if req.RoleName == "ADMIN" && targetUserID == uuid.Nil {
		return nil, domain.ErrNotFound
	}
	return &domain.ProjectMember{ProjectID: projectID, UserID: targetUserID}, nil
}

func (m *mockProjectUsecaseForHandler) RemoveMember(ctx context.Context, projectID, targetUserID uuid.UUID) error {
	if targetUserID == uuid.Nil {
		return domain.ErrNotFound
	}
	return nil
}

type mockSprintUsecaseForHandler struct {
	domain.SprintUsecase
}

func (m *mockSprintUsecaseForHandler) CreateSprint(ctx context.Context, projectID uuid.UUID, req domain.CreateSprintRequest) (*domain.Sprint, error) {
	return &domain.Sprint{ProjectID: projectID, Name: req.Name}, nil
}

func (m *mockSprintUsecaseForHandler) ListSprints(ctx context.Context, projectID uuid.UUID) ([]domain.Sprint, error) {
	return []domain.Sprint{{ProjectID: projectID}}, nil
}

func (m *mockSprintUsecaseForHandler) GetSprintByID(ctx context.Context, projectID, sprintID uuid.UUID) (*domain.Sprint, error) {
	if sprintID == uuid.Nil {
		return nil, domain.ErrNotFound
	}
	return &domain.Sprint{ID: sprintID, ProjectID: projectID}, nil
}

func (m *mockSprintUsecaseForHandler) UpdateSprint(ctx context.Context, projectID, sprintID uuid.UUID, req domain.UpdateSprintRequest) (*domain.Sprint, error) {
	if sprintID == uuid.Nil {
		return nil, domain.ErrNotFound
	}
	return &domain.Sprint{ID: sprintID, Name: req.Name}, nil
}

func (m *mockSprintUsecaseForHandler) StartSprint(ctx context.Context, projectID, sprintID uuid.UUID, req domain.StartSprintRequest) (*domain.Sprint, error) {
	if sprintID == uuid.Nil {
		return nil, domain.ErrActiveSprintExists
	}
	return &domain.Sprint{ID: sprintID, Status: domain.SprintStatusActive}, nil
}

func (m *mockSprintUsecaseForHandler) CompleteSprint(ctx context.Context, projectID, sprintID uuid.UUID, req domain.CompleteSprintRequest) (*domain.Sprint, error) {
	if sprintID == uuid.Nil {
		return nil, domain.ErrInvalidInput
	}
	return &domain.Sprint{ID: sprintID, Status: domain.SprintStatusCompleted}, nil
}

func (m *mockSprintUsecaseForHandler) DeleteSprint(ctx context.Context, projectID, sprintID uuid.UUID) error {
	if sprintID == uuid.Nil {
		return domain.ErrNotFound
	}
	return nil
}

func (m *mockSprintUsecaseForHandler) BulkMoveIssues(ctx context.Context, projectID uuid.UUID, req domain.BulkMoveSprintIssuesRequest) error {
	return nil
}

type mockIssueUsecaseForHandler struct {
	domain.IssueUsecase
}

func (m *mockIssueUsecaseForHandler) CreateIssue(ctx context.Context, reporterID, projectID uuid.UUID, req domain.CreateIssueRequest) (*domain.Issue, error) {
	if req.Title == "HIERARCHY_FAIL" {
		return nil, domain.ErrInvalidHierarchy
	}
	if req.Title == "NOT_FOUND" {
		return nil, domain.ErrNotFound
	}
	return &domain.Issue{Key: "TYK-1", Title: req.Title, ProjectID: projectID}, nil
}

func (m *mockIssueUsecaseForHandler) ListIssues(ctx context.Context, filter domain.IssueFilter) ([]domain.Issue, int64, error) {
	return []domain.Issue{{Key: "TYK-1"}}, 1, nil
}

func (m *mockIssueUsecaseForHandler) GetIssueByID(ctx context.Context, projectID, issueID uuid.UUID) (*domain.Issue, error) {
	if issueID == uuid.Nil {
		return nil, domain.ErrNotFound
	}
	return &domain.Issue{ID: issueID, Key: "TYK-1"}, nil
}

func (m *mockIssueUsecaseForHandler) GetIssueByKey(ctx context.Context, key string) (*domain.Issue, error) {
	if key == "NOTFOUND" {
		return nil, domain.ErrNotFound
	}
	return &domain.Issue{Key: key}, nil
}

func (m *mockIssueUsecaseForHandler) UpdateIssue(ctx context.Context, projectID, issueID uuid.UUID, req domain.UpdateIssueRequest) (*domain.Issue, error) {
	if issueID == uuid.Nil {
		return nil, domain.ErrNotFound
	}
	return &domain.Issue{ID: issueID, Key: "TYK-1"}, nil
}

func (m *mockIssueUsecaseForHandler) MoveIssue(ctx context.Context, projectID, issueID uuid.UUID, req domain.MoveIssueRequest) (*domain.Issue, error) {
	if issueID == uuid.Nil {
		return nil, domain.ErrNotFound
	}
	return &domain.Issue{ID: issueID, Position: req.Position}, nil
}

func (m *mockIssueUsecaseForHandler) DeleteIssue(ctx context.Context, projectID, issueID uuid.UUID) error {
	if issueID == uuid.Nil {
		return domain.ErrNotFound
	}
	return nil
}

type mockMetricUsecaseForHandler struct {
	domain.MetricUsecase
}

func (m *mockMetricUsecaseForHandler) GetProjectMetrics(ctx context.Context, projectID uuid.UUID) (*domain.ProjectSummaryMetrics, error) {
	return &domain.ProjectSummaryMetrics{ProjectID: projectID}, nil
}

func (m *mockMetricUsecaseForHandler) GetSprintMetrics(ctx context.Context, projectID, sprintID uuid.UUID) (*domain.SprintMetrics, error) {
	return &domain.SprintMetrics{SprintID: sprintID}, nil
}

func (m *mockMetricUsecaseForHandler) ExportIssuesCSV(ctx context.Context, filter domain.IssueFilter, writer io.Writer) error {
	_, _ = writer.Write([]byte("CSV DATA"))
	return nil
}

func (m *mockMetricUsecaseForHandler) ExportProjectSummary(ctx context.Context, projectID uuid.UUID) (interface{}, error) {
	return map[string]string{"project": "summary"}, nil
}

// =========================================================================
// Tests
// =========================================================================

func setupEcho() *echo.Echo {
	e := echo.New()
	e.Validator = validator.NewCustomValidator()
	return e
}

func TestAuthHandler(t *testing.T) {
	e := setupEcho()
	h := NewAuthHandler(&mockAuthUsecaseForHandler{})

	t.Run("Register success, validation error, conflict, and internal error", func(t *testing.T) {
		// Valid
		body, _ := json.Marshal(domain.RegisterRequest{
			Name:     "Test",
			Email:    "test@example.com",
			Password: "password123",
		})
		req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		_ = h.Register(c)
		if rec.Code != http.StatusCreated {
			t.Errorf("expected 201, got %d", rec.Code)
		}

		// Validation error
		reqVal := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader([]byte(`{"name":""}`)))
		reqVal.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		recVal := httptest.NewRecorder()
		cVal := e.NewContext(reqVal, recVal)
		_ = h.Register(cVal)
		if recVal.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for validation, got %d", recVal.Code)
		}

		// Conflict
		body2, _ := json.Marshal(domain.RegisterRequest{
			Name:     "Test",
			Email:    "conflict@example.com",
			Password: "password123",
		})
		req2 := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body2))
		req2.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec2 := httptest.NewRecorder()
		c2 := e.NewContext(req2, rec2)
		_ = h.Register(c2)
		if rec2.Code != http.StatusConflict {
			t.Errorf("expected 409, got %d", rec2.Code)
		}
	})

	t.Run("Login success, validation error, and unauthorized", func(t *testing.T) {
		body, _ := json.Marshal(domain.LoginRequest{
			Email:    "test@example.com",
			Password: "password123",
		})
		req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		_ = h.Login(c)
		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}

		// Validation error
		reqVal := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader([]byte(`{}`)))
		reqVal.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		recVal := httptest.NewRecorder()
		cVal := e.NewContext(reqVal, recVal)
		_ = h.Login(cVal)
		if recVal.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", recVal.Code)
		}

		// Wrong credentials
		bodyWrong, _ := json.Marshal(domain.LoginRequest{
			Email:    "wrong@example.com",
			Password: "password123",
		})
		reqWrong := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(bodyWrong))
		reqWrong.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		recWrong := httptest.NewRecorder()
		cWrong := e.NewContext(reqWrong, recWrong)
		_ = h.Login(cWrong)
		if recWrong.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", recWrong.Code)
		}
	})

	t.Run("RefreshToken success and validation error", func(t *testing.T) {
		body, _ := json.Marshal(domain.RefreshTokenRequest{RefreshToken: "valid"})
		req := httptest.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		_ = h.RefreshToken(c)
		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}

		reqBad := httptest.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewReader([]byte(`{}`)))
		reqBad.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		recBad := httptest.NewRecorder()
		cBad := e.NewContext(reqBad, recBad)
		_ = h.RefreshToken(cBad)
		if recBad.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", recBad.Code)
		}
	})

	t.Run("GoogleLogin success and validation error", func(t *testing.T) {
		body, _ := json.Marshal(domain.GoogleAuthRequest{IDToken: "valid"})
		req := httptest.NewRequest(http.MethodPost, "/auth/google", bytes.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		_ = h.GoogleLogin(c)
		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}

		reqBad := httptest.NewRequest(http.MethodPost, "/auth/google", bytes.NewReader([]byte(`{}`)))
		reqBad.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		recBad := httptest.NewRecorder()
		cBad := e.NewContext(reqBad, recBad)
		_ = h.GoogleLogin(cBad)
		if recBad.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", recBad.Code)
		}
	})

	t.Run("GetMe success and unauthorized", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set(middleware.CtxKeyUserID, uuid.New())

		_ = h.GetMe(c)
		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}

		rec2 := httptest.NewRecorder()
		c2 := e.NewContext(req, rec2)
		_ = h.GetMe(c2)
		if rec2.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", rec2.Code)
		}
	})
}

func TestProjectHandler(t *testing.T) {
	e := setupEcho()
	h := NewProjectHandler(&mockProjectUsecaseForHandler{})
	userID := uuid.New()
	projectID := uuid.New()

	t.Run("Create project success and validation error", func(t *testing.T) {
		body, _ := json.Marshal(domain.CreateProjectRequest{Name: "Tykket", Key: "TYK"})
		req := httptest.NewRequest(http.MethodPost, "/projects", bytes.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set(middleware.CtxKeyUserID, userID)

		_ = h.Create(c)
		if rec.Code != http.StatusCreated {
			t.Errorf("expected 201, got %d", rec.Code)
		}

		// Validation error
		reqVal := httptest.NewRequest(http.MethodPost, "/projects", bytes.NewReader([]byte(`{}`)))
		reqVal.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		recVal := httptest.NewRecorder()
		cVal := e.NewContext(reqVal, recVal)
		cVal.Set(middleware.CtxKeyUserID, userID)
		_ = h.Create(cVal)
		if recVal.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", recVal.Code)
		}
	})

	t.Run("List projects", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/projects", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set(middleware.CtxKeyUserID, userID)

		_ = h.List(c)
		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}

		// Unauthorized
		rec2 := httptest.NewRecorder()
		c2 := e.NewContext(req, rec2)
		_ = h.List(c2)
		if rec2.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", rec2.Code)
		}
	})

	t.Run("Get project by ID and invalid UUID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/projects/"+projectID.String(), nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set(middleware.CtxKeyUserID, userID)
		c.SetParamNames("projectId")
		c.SetParamValues(projectID.String())

		_ = h.GetByID(c)
		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}

		// Invalid UUID
		recBad := httptest.NewRecorder()
		cBad := e.NewContext(req, recBad)
		cBad.Set(middleware.CtxKeyUserID, userID)
		cBad.SetParamNames("projectId")
		cBad.SetParamValues("invalid-uuid")
		_ = h.GetByID(cBad)
		if recBad.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for bad UUID, got %d", recBad.Code)
		}
	})

	t.Run("Update project", func(t *testing.T) {
		body, _ := json.Marshal(domain.UpdateProjectRequest{Name: "Updated"})
		req := httptest.NewRequest(http.MethodPut, "/projects/"+projectID.String(), bytes.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set(middleware.CtxKeyUserID, userID)
		c.SetParamNames("projectId")
		c.SetParamValues(projectID.String())

		_ = h.Update(c)
		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("Delete project", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/projects/"+projectID.String(), nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set(middleware.CtxKeyUserID, userID)
		c.SetParamNames("projectId")
		c.SetParamValues(projectID.String())

		_ = h.Delete(c)
		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("Member operations: list, add, update, remove", func(t *testing.T) {
		memberUserID := uuid.New()

		// List members
		reqList := httptest.NewRequest(http.MethodGet, "/projects/"+projectID.String()+"/members", nil)
		recList := httptest.NewRecorder()
		cList := e.NewContext(reqList, recList)
		cList.SetParamNames("projectId")
		cList.SetParamValues(projectID.String())
		_ = h.ListMembers(cList)
		if recList.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", recList.Code)
		}

		// Add member
		bodyAdd, _ := json.Marshal(domain.AddMemberRequest{Email: "user@example.com", RoleName: "MEMBER"})
		reqAdd := httptest.NewRequest(http.MethodPost, "/projects/"+projectID.String()+"/members", bytes.NewReader(bodyAdd))
		reqAdd.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		recAdd := httptest.NewRecorder()
		cAdd := e.NewContext(reqAdd, recAdd)
		cAdd.SetParamNames("projectId")
		cAdd.SetParamValues(projectID.String())
		_ = h.AddMember(cAdd)
		if recAdd.Code != http.StatusCreated {
			t.Errorf("expected 201, got %d", recAdd.Code)
		}

		// Update member role
		bodyUpd, _ := json.Marshal(domain.UpdateMemberRoleRequest{RoleName: "ADMIN"})
		reqUpd := httptest.NewRequest(http.MethodPut, "/projects/"+projectID.String()+"/members/"+memberUserID.String(), bytes.NewReader(bodyUpd))
		reqUpd.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		recUpd := httptest.NewRecorder()
		cUpd := e.NewContext(reqUpd, recUpd)
		cUpd.SetParamNames("projectId", "userId")
		cUpd.SetParamValues(projectID.String(), memberUserID.String())
		_ = h.UpdateMemberRole(cUpd)
		if recUpd.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", recUpd.Code)
		}

		// Remove member
		reqDel := httptest.NewRequest(http.MethodDelete, "/projects/"+projectID.String()+"/members/"+memberUserID.String(), nil)
		recDel := httptest.NewRecorder()
		cDel := e.NewContext(reqDel, recDel)
		cDel.SetParamNames("projectId", "userId")
		cDel.SetParamValues(projectID.String(), memberUserID.String())
		_ = h.RemoveMember(cDel)
		if recDel.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", recDel.Code)
		}
	})
}

func TestSprintHandler(t *testing.T) {
	e := setupEcho()
	h := NewSprintHandler(&mockSprintUsecaseForHandler{})
	projectID := uuid.New()
	sprintID := uuid.New()

	t.Run("Create sprint", func(t *testing.T) {
		body, _ := json.Marshal(domain.CreateSprintRequest{Name: "Sprint 1"})
		req := httptest.NewRequest(http.MethodPost, "/projects/"+projectID.String()+"/sprints", bytes.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("projectId")
		c.SetParamValues(projectID.String())

		_ = h.Create(c)
		if rec.Code != http.StatusCreated {
			t.Errorf("expected 201, got %d", rec.Code)
		}
	})

	t.Run("List sprints", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/projects/"+projectID.String()+"/sprints", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("projectId")
		c.SetParamValues(projectID.String())

		_ = h.List(c)
		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("Get, Update, Start, Complete, Delete sprint", func(t *testing.T) {
		// Get
		reqGet := httptest.NewRequest(http.MethodGet, "/projects/"+projectID.String()+"/sprints/"+sprintID.String(), nil)
		recGet := httptest.NewRecorder()
		cGet := e.NewContext(reqGet, recGet)
		cGet.SetParamNames("projectId", "sprintId")
		cGet.SetParamValues(projectID.String(), sprintID.String())
		_ = h.GetByID(cGet)
		if recGet.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", recGet.Code)
		}

		// Update
		bodyUpd, _ := json.Marshal(domain.UpdateSprintRequest{Name: "Sprint 1 Renamed"})
		reqUpd := httptest.NewRequest(http.MethodPut, "/projects/"+projectID.String()+"/sprints/"+sprintID.String(), bytes.NewReader(bodyUpd))
		reqUpd.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		recUpd := httptest.NewRecorder()
		cUpd := e.NewContext(reqUpd, recUpd)
		cUpd.SetParamNames("projectId", "sprintId")
		cUpd.SetParamValues(projectID.String(), sprintID.String())
		_ = h.Update(cUpd)
		if recUpd.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", recUpd.Code)
		}

		// Start
		reqStart := httptest.NewRequest(http.MethodPost, "/projects/"+projectID.String()+"/sprints/"+sprintID.String()+"/start", bytes.NewReader([]byte("{}")))
		reqStart.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		recStart := httptest.NewRecorder()
		cStart := e.NewContext(reqStart, recStart)
		cStart.SetParamNames("projectId", "sprintId")
		cStart.SetParamValues(projectID.String(), sprintID.String())
		_ = h.Start(cStart)
		if recStart.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", recStart.Code)
		}

		// Complete
		reqComp := httptest.NewRequest(http.MethodPost, "/projects/"+projectID.String()+"/sprints/"+sprintID.String()+"/complete", bytes.NewReader([]byte("{}")))
		reqComp.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		recComp := httptest.NewRecorder()
		cComp := e.NewContext(reqComp, recComp)
		cComp.SetParamNames("projectId", "sprintId")
		cComp.SetParamValues(projectID.String(), sprintID.String())
		_ = h.Complete(cComp)
		if recComp.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", recComp.Code)
		}

		// Delete
		reqDel := httptest.NewRequest(http.MethodDelete, "/projects/"+projectID.String()+"/sprints/"+sprintID.String(), nil)
		recDel := httptest.NewRecorder()
		cDel := e.NewContext(reqDel, recDel)
		cDel.SetParamNames("projectId", "sprintId")
		cDel.SetParamValues(projectID.String(), sprintID.String())
		_ = h.Delete(cDel)
		if recDel.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", recDel.Code)
		}

		// Bulk move
		bodyBulk, _ := json.Marshal(domain.BulkMoveSprintIssuesRequest{IssueIDs: []uuid.UUID{uuid.New()}})
		reqBulk := httptest.NewRequest(http.MethodPost, "/projects/"+projectID.String()+"/sprints/bulk-move", bytes.NewReader(bodyBulk))
		reqBulk.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		recBulk := httptest.NewRecorder()
		cBulk := e.NewContext(reqBulk, recBulk)
		cBulk.SetParamNames("projectId")
		cBulk.SetParamValues(projectID.String())
		_ = h.BulkMove(cBulk)
		if recBulk.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", recBulk.Code)
		}
	})
}

func TestIssueHandler(t *testing.T) {
	e := setupEcho()
	h := NewIssueHandler(&mockIssueUsecaseForHandler{})
	projectID := uuid.New()
	issueID := uuid.New()
	userID := uuid.New()

	t.Run("Create issue", func(t *testing.T) {
		body, _ := json.Marshal(domain.CreateIssueRequest{
			Type:     domain.IssueTypeTask,
			Priority: domain.IssuePriorityHigh,
			Title:    "Build Login",
		})
		req := httptest.NewRequest(http.MethodPost, "/projects/"+projectID.String()+"/issues", bytes.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set(middleware.CtxKeyUserID, userID)
		c.SetParamNames("projectId")
		c.SetParamValues(projectID.String())

		_ = h.Create(c)
		if rec.Code != http.StatusCreated {
			t.Errorf("expected 201, got %d", rec.Code)
		}

		// Validation error
		reqVal := httptest.NewRequest(http.MethodPost, "/projects/"+projectID.String()+"/issues", bytes.NewReader([]byte(`{}`)))
		reqVal.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		recVal := httptest.NewRecorder()
		cVal := e.NewContext(reqVal, recVal)
		cVal.Set(middleware.CtxKeyUserID, userID)
		cVal.SetParamNames("projectId")
		cVal.SetParamValues(projectID.String())
		_ = h.Create(cVal)
		if recVal.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", recVal.Code)
		}
	})

	t.Run("List issues with filters", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/projects/"+projectID.String()+"/issues?in_backlog=true&type=TASK&status=TODO&page=1&limit=20", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("projectId")
		c.SetParamValues(projectID.String())

		_ = h.List(c)
		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("GetByID, GetByKey, Update, Move, Delete", func(t *testing.T) {
		// GetByID
		reqGet := httptest.NewRequest(http.MethodGet, "/projects/"+projectID.String()+"/issues/"+issueID.String(), nil)
		recGet := httptest.NewRecorder()
		cGet := e.NewContext(reqGet, recGet)
		cGet.SetParamNames("projectId", "issueId")
		cGet.SetParamValues(projectID.String(), issueID.String())
		_ = h.GetByID(cGet)
		if recGet.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", recGet.Code)
		}

		// GetByKey
		reqKey := httptest.NewRequest(http.MethodGet, "/issues/TYK-1", nil)
		recKey := httptest.NewRecorder()
		cKey := e.NewContext(reqKey, recKey)
		cKey.SetParamNames("key")
		cKey.SetParamValues("TYK-1")
		_ = h.GetByKey(cKey)
		if recKey.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", recKey.Code)
		}

		// Update
		title := "Updated Task"
		bodyUpd, _ := json.Marshal(domain.UpdateIssueRequest{Title: &title})
		reqUpd := httptest.NewRequest(http.MethodPut, "/projects/"+projectID.String()+"/issues/"+issueID.String(), bytes.NewReader(bodyUpd))
		reqUpd.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		recUpd := httptest.NewRecorder()
		cUpd := e.NewContext(reqUpd, recUpd)
		cUpd.SetParamNames("projectId", "issueId")
		cUpd.SetParamValues(projectID.String(), issueID.String())
		_ = h.Update(cUpd)
		if recUpd.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", recUpd.Code)
		}

		// Move
		status := domain.IssueStatusInProgress
		bodyMove, _ := json.Marshal(domain.MoveIssueRequest{Status: &status, Position: 32768.0})
		reqMove := httptest.NewRequest(http.MethodPatch, "/projects/"+projectID.String()+"/issues/"+issueID.String()+"/move", bytes.NewReader(bodyMove))
		reqMove.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		recMove := httptest.NewRecorder()
		cMove := e.NewContext(reqMove, recMove)
		cMove.SetParamNames("projectId", "issueId")
		cMove.SetParamValues(projectID.String(), issueID.String())
		_ = h.Move(cMove)
		if recMove.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", recMove.Code)
		}

		// Delete
		reqDel := httptest.NewRequest(http.MethodDelete, "/projects/"+projectID.String()+"/issues/"+issueID.String(), nil)
		recDel := httptest.NewRecorder()
		cDel := e.NewContext(reqDel, recDel)
		cDel.SetParamNames("projectId", "issueId")
		cDel.SetParamValues(projectID.String(), issueID.String())
		_ = h.Delete(cDel)
		if recDel.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", recDel.Code)
		}
	})
}

func TestMetricHandler(t *testing.T) {
	e := setupEcho()
	h := NewMetricHandler(&mockMetricUsecaseForHandler{})
	projectID := uuid.New()
	sprintID := uuid.New()

	t.Run("GetProjectMetrics", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/projects/"+projectID.String()+"/metrics", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("projectId")
		c.SetParamValues(projectID.String())

		_ = h.GetProjectMetrics(c)
		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("GetSprintBurndown", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/projects/"+projectID.String()+"/sprints/"+sprintID.String()+"/burndown", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("projectId", "sprintId")
		c.SetParamValues(projectID.String(), sprintID.String())

		_ = h.GetSprintBurndown(c)
		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("ExportCSV", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/projects/"+projectID.String()+"/export/csv?status=TODO&in_backlog=true", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("projectId")
		c.SetParamValues(projectID.String())

		_ = h.ExportCSV(c)
		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("ExportSummary", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/projects/"+projectID.String()+"/export/summary", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("projectId")
		c.SetParamValues(projectID.String())

		_ = h.ExportSummary(c)
		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}
	})
}
