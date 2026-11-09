package usecase

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/luqmanulhakimdev/tykket-go/internal/domain"
	"github.com/luqmanulhakimdev/tykket-go/pkg/token"
)

type failUserRepo struct {
	domain.UserRepository
	err error
}

func (m *failUserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	return nil, m.err
}

func (m *failUserRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	return nil, m.err
}

func (m *failUserRepo) Create(ctx context.Context, user *domain.User) error {
	return m.err
}

func (m *failUserRepo) GetByProvider(ctx context.Context, provider domain.AuthProvider, providerID string) (*domain.User, error) {
	return nil, m.err
}

type failTokenMgr struct {
	token.TokenManager
	err error
}

func (m *failTokenMgr) GenerateTokenPair(userID uuid.UUID, email string) (*token.TokenPair, error) {
	return nil, m.err
}

func (m *failTokenMgr) ValidateToken(tokenString string, expectedType token.TokenType) (*token.Claims, error) {
	return nil, m.err
}

type failSprintRepo struct {
	domain.SprintRepository
	err error
}

func (m *failSprintRepo) Create(ctx context.Context, s *domain.Sprint) error {
	return m.err
}

func (m *failSprintRepo) Update(ctx context.Context, s *domain.Sprint) error {
	return m.err
}

func (m *failSprintRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return m.err
}

func (m *failSprintRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Sprint, error) {
	return nil, m.err
}

func (m *failSprintRepo) GetActiveByProjectID(ctx context.Context, projectID uuid.UUID) (*domain.Sprint, error) {
	return nil, m.err
}

type failIssueRepo struct {
	domain.IssueRepository
	err error
}

func (m *failIssueRepo) Create(ctx context.Context, issue *domain.Issue) error {
	return m.err
}

func (m *failIssueRepo) Update(ctx context.Context, issue *domain.Issue) error {
	return m.err
}

func (m *failIssueRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return m.err
}

func (m *failIssueRepo) UpdatePositionAndStatus(ctx context.Context, issueID uuid.UUID, status *domain.IssueStatus, pos float64, sprintID *uuid.UUID) error {
	return m.err
}

func (m *failIssueRepo) MoveIncompleteIssues(ctx context.Context, from uuid.UUID, to *uuid.UUID) error {
	return m.err
}

func (m *failIssueRepo) BulkMoveIssues(ctx context.Context, ids []uuid.UUID, sID *uuid.UUID) error {
	return m.err
}

func (m *failIssueRepo) List(ctx context.Context, filter domain.IssueFilter) ([]domain.Issue, int64, error) {
	return nil, 0, m.err
}

type failProjectRepo struct {
	domain.ProjectRepository
	err error
}

func (m *failProjectRepo) IncrementIssueCounter(ctx context.Context, projectID uuid.UUID) (string, int64, error) {
	return "", 0, m.err
}

func (m *failProjectRepo) Update(ctx context.Context, p *domain.Project) error {
	return m.err
}

func (m *failProjectRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Project, error) {
	return nil, m.err
}

type failRBACRepo struct {
	domain.RBACRepository
	err error
}

func (m *failRBACRepo) GetRoleByName(ctx context.Context, name string) (*domain.Role, error) {
	return nil, m.err
}

func (m *failRBACRepo) AddProjectMember(ctx context.Context, member *domain.ProjectMember) error {
	return m.err
}

func (m *failRBACRepo) UpdateProjectMemberRole(ctx context.Context, pID, uID, rID uuid.UUID) error {
	return m.err
}

type failMetricRepo struct {
	domain.MetricRepository
	err error
}

func (m *failMetricRepo) GetProjectMetrics(ctx context.Context, projectID uuid.UUID) (*domain.ProjectSummaryMetrics, error) {
	return nil, m.err
}

func TestAuthUsecase_ExhaustiveErrors(t *testing.T) {
	ctx := context.Background()
	dbErr := errors.New("db error")

	// 1. Register: GetByEmail fails with DB error
	uc1 := NewAuthUsecase(&failUserRepo{err: dbErr}, &mockTokenMgrForAuth{})
	_, err := uc1.Register(ctx, domain.RegisterRequest{Email: "a@b.com", Name: "A", Password: "pwd"})
	if err == nil {
		t.Error("expected error for Register GetByEmail failure")
	}

	// 2. Register: Create fails
	fakeRepo := &mockUserRepoForAuth{
		usersByEmail: make(map[string]*domain.User),
		usersByID:    make(map[uuid.UUID]*domain.User),
	}
	uc2 := NewAuthUsecase(&failUserRepo{UserRepository: fakeRepo, err: dbErr}, &mockTokenMgrForAuth{})
	_, err = uc2.Register(ctx, domain.RegisterRequest{Email: "a@b.com", Name: "A", Password: "pwd"})
	if err == nil {
		t.Error("expected error for Register Create failure")
	}

	// 3. Register: GenerateTokenPair fails
	uc3 := NewAuthUsecase(fakeRepo, &failTokenMgr{err: errors.New("token err")})
	_, err = uc3.Register(ctx, domain.RegisterRequest{Email: "c@d.com", Name: "C", Password: "pwd"})
	if err == nil {
		t.Error("expected error for Register GenerateTokenPair failure")
	}

	// 4. Login: GetByEmail generic DB error
	uc4 := NewAuthUsecase(&failUserRepo{err: dbErr}, &mockTokenMgrForAuth{})
	_, err = uc4.Login(ctx, domain.LoginRequest{Email: "a@b.com", Password: "pwd"})
	if err == nil || errors.Is(err, domain.ErrUnauthorized) {
		t.Error("expected non-unauthorized DB error for Login")
	}

	// 5. RefreshToken: GetByID fails
	uc5 := NewAuthUsecase(&failUserRepo{err: domain.ErrNotFound}, &mockTokenMgrForAuth{})
	_, err = uc5.RefreshToken(ctx, domain.RefreshTokenRequest{RefreshToken: "valid-refresh-token"})
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized when user not found in refresh, got %v", err)
	}

	// 6. RefreshToken: Token generation error
	validUser := &domain.User{ID: uuid.MustParse("00000000-0000-0000-0000-000000000001"), Email: "test@example.com"}
	repoWithUser := &mockUserRepoForAuth{
		usersByEmail: map[string]*domain.User{"test@example.com": validUser},
		usersByID:    map[uuid.UUID]*domain.User{validUser.ID: validUser},
	}
	uc6 := NewAuthUsecase(repoWithUser, &failTokenMgr{err: errors.New("token err")})
	_, err = uc6.RefreshToken(ctx, domain.RefreshTokenRequest{RefreshToken: "valid-refresh-token"})
	if err == nil {
		t.Error("expected error when token generation fails during refresh")
	}

	// 7. GoogleLogin: GetByProvider DB error
	uc7 := NewAuthUsecase(&failUserRepo{err: dbErr}, &mockTokenMgrForAuth{})
	_, err = uc7.GoogleLogin(ctx, domain.GoogleAuthRequest{IDToken: "token"})
	if err == nil {
		t.Error("expected error for GoogleLogin DB error")
	}

	// 8. GoogleLogin: Existing user (second login)
	provID := "g-123456"
	existingGoogleUser := &domain.User{
		ID:           uuid.New(),
		Email:        "google@example.com",
		AuthProvider: domain.AuthProviderGoogle,
		ProviderID:   &provID,
	}
	gRepo := &mockUserRepoForAuth{
		usersByID:    map[uuid.UUID]*domain.User{existingGoogleUser.ID: existingGoogleUser},
		usersByEmail: map[string]*domain.User{"google@example.com": existingGoogleUser},
	}
	uc8 := NewAuthUsecase(gRepo, &mockTokenMgrForAuth{})
	res8, err := uc8.GoogleLogin(ctx, domain.GoogleAuthRequest{IDToken: "token"})
	if err != nil || res8.User.Email != "google@example.com" {
		t.Errorf("expected existing user returned from GoogleLogin, got %v", err)
	}
}

func TestSprintUsecase_ExhaustiveErrors(t *testing.T) {
	ctx := context.Background()
	projectID := uuid.New()
	sprintID := uuid.New()
	dbErr := errors.New("db error")

	// 1. CreateSprint repo error
	uc1 := NewSprintUsecase(&failSprintRepo{err: dbErr}, &mockIssueRepoForSprintUC{})
	_, err := uc1.CreateSprint(ctx, projectID, domain.CreateSprintRequest{Name: "S"})
	if err == nil {
		t.Error("expected error on CreateSprint")
	}

	// 2. StartSprint with non-nil StartDate & EndDate, and Update repo error
	now := time.Now()
	later := now.Add(7 * 24 * time.Hour)
	baseSprint := &domain.Sprint{
		ID:        sprintID,
		ProjectID: projectID,
		Status:    domain.SprintStatusPlanned,
	}
	mockSprintRepo := &mockSprintRepoForSprintUC{
		sprints: map[uuid.UUID]*domain.Sprint{sprintID: baseSprint},
	}
	uc2 := NewSprintUsecase(mockSprintRepo, &mockIssueRepoForSprintUC{})
	started, err := uc2.StartSprint(ctx, projectID, sprintID, domain.StartSprintRequest{
		StartDate: &now,
		EndDate:   &later,
	})
	if err != nil || started.EndDate == nil {
		t.Fatalf("expected started with explicit dates, got %v", err)
	}

	// 3. StartSprint GetActiveByProjectID error
	uc3 := NewSprintUsecase(&failSprintRepo{err: dbErr}, &mockIssueRepoForSprintUC{})
	_, err = uc3.StartSprint(ctx, projectID, sprintID, domain.StartSprintRequest{})
	if err == nil {
		t.Error("expected error on StartSprint when GetByID fails")
	}

	// 4. CompleteSprint MoveIncompleteIssues failure
	mockSprintRepo2 := &mockSprintRepoForSprintUC{
		sprints: map[uuid.UUID]*domain.Sprint{
			sprintID: {ID: sprintID, ProjectID: projectID, Status: domain.SprintStatusActive},
		},
	}
	uc4 := NewSprintUsecase(mockSprintRepo2, &failIssueRepo{err: dbErr})
	_, err = uc4.CompleteSprint(ctx, projectID, sprintID, domain.CompleteSprintRequest{})
	if err == nil {
		t.Error("expected error on CompleteSprint reconcile failure")
	}

	// 5. DeleteSprint failure
	uc5 := NewSprintUsecase(&failSprintRepo{err: dbErr}, &mockIssueRepoForSprintUC{})
	err = uc5.DeleteSprint(ctx, projectID, sprintID)
	if err == nil {
		t.Error("expected error on DeleteSprint")
	}

	// 6. BulkMoveIssues repo failure
	uc6 := NewSprintUsecase(mockSprintRepo2, &failIssueRepo{err: dbErr})
	err = uc6.BulkMoveIssues(ctx, projectID, domain.BulkMoveSprintIssuesRequest{
		IssueIDs: []uuid.UUID{uuid.New()},
	})
	if err == nil {
		t.Error("expected error on BulkMoveIssues")
	}
}

func strVal(s string) *string { return &s }

func TestIssueUsecase_ExhaustiveErrors(t *testing.T) {
	ctx := context.Background()
	projectID := uuid.New()
	issueID := uuid.New()
	sprintID := uuid.New()
	dbErr := errors.New("db error")

	// 1. CreateIssue: sprint not found
	uc1 := NewIssueUsecase(&mockIssueRepo{issues: make(map[uuid.UUID]*domain.Issue)}, &mockProjectRepo{}, &failSprintRepo{err: domain.ErrNotFound})
	_, err := uc1.CreateIssue(ctx, uuid.New(), projectID, domain.CreateIssueRequest{
		Type:     domain.IssueTypeTask,
		Title:    "T",
		SprintID: &sprintID,
	})
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound for missing sprint, got %v", err)
	}

	// 2. CreateIssue: sprint belongs to other project -> ErrForbidden
	otherProjectID := uuid.New()
	sprintOtherProj := &mockSprintRepoForSprintUC{
		sprints: map[uuid.UUID]*domain.Sprint{
			sprintID: {ID: sprintID, ProjectID: otherProjectID},
		},
	}
	uc2 := NewIssueUsecase(&mockIssueRepo{issues: make(map[uuid.UUID]*domain.Issue)}, &mockProjectRepo{}, sprintOtherProj)
	_, err = uc2.CreateIssue(ctx, uuid.New(), projectID, domain.CreateIssueRequest{
		Type:     domain.IssueTypeTask,
		Title:    "T",
		SprintID: &sprintID,
	})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("expected ErrForbidden for sprint in other project, got %v", err)
	}

	// 3. CreateIssue: IncrementIssueCounter error
	uc3 := NewIssueUsecase(&mockIssueRepo{issues: make(map[uuid.UUID]*domain.Issue)}, &failProjectRepo{err: dbErr}, &mockSprintRepoForSprintUC{})
	_, err = uc3.CreateIssue(ctx, uuid.New(), projectID, domain.CreateIssueRequest{
		Type:  domain.IssueTypeTask,
		Title: "T",
	})
	if err == nil {
		t.Error("expected error on IncrementIssueCounter failure")
	}

	// 4. CreateIssue: issueRepo.Create error
	uc4 := NewIssueUsecase(&failIssueRepo{err: dbErr}, &mockProjectRepo{}, &mockSprintRepoForSprintUC{})
	_, err = uc4.CreateIssue(ctx, uuid.New(), projectID, domain.CreateIssueRequest{
		Type:  domain.IssueTypeTask,
		Title: "T",
	})
	if err == nil {
		t.Error("expected error on issueRepo.Create failure")
	}

	// 5. UpdateIssue: issueRepo.Update error
	baseIssue := &domain.Issue{
		ID:        issueID,
		ProjectID: projectID,
		Type:      domain.IssueTypeTask,
		Title:     "Old",
	}
	issueMap := map[uuid.UUID]*domain.Issue{issueID: baseIssue}
	uc5 := NewIssueUsecase(&failIssueRepo{IssueRepository: &mockIssueRepo{issues: issueMap}, err: dbErr}, &mockProjectRepo{}, &mockSprintRepoForSprintUC{})
	_, err = uc5.UpdateIssue(ctx, projectID, issueID, domain.UpdateIssueRequest{
		Title: strVal("New Title"),
	})
	if err == nil {
		t.Error("expected error on UpdateIssue failure")
	}

	// 6. MoveIssue: UpdatePositionAndStatus error
	uc6 := NewIssueUsecase(&failIssueRepo{IssueRepository: &mockIssueRepo{issues: issueMap}, err: dbErr}, &mockProjectRepo{}, &mockSprintRepoForSprintUC{})
	_, err = uc6.MoveIssue(ctx, projectID, issueID, domain.MoveIssueRequest{Position: 100})
	if err == nil {
		t.Error("expected error on MoveIssue failure")
	}

	// 7. DeleteIssue: Delete error
	uc7 := NewIssueUsecase(&failIssueRepo{IssueRepository: &mockIssueRepo{issues: issueMap}, err: dbErr}, &mockProjectRepo{}, &mockSprintRepoForSprintUC{})
	err = uc7.DeleteIssue(ctx, projectID, issueID)
	if err == nil {
		t.Error("expected error on DeleteIssue failure")
	}
}

func TestProjectUsecase_ExhaustiveErrors(t *testing.T) {
	ctx := context.Background()
	ownerID := uuid.New()
	notOwnerID := uuid.New()
	projectID := uuid.New()
	dbErr := errors.New("db error")

	project := &domain.Project{
		ID:      projectID,
		OwnerID: ownerID,
		Name:    "Project A",
	}

	// 1. DeleteProject by non-owner -> ErrForbidden
	mockProjRepo := &mockProjectRepoForProjectUC{
		projectsByID: map[uuid.UUID]*domain.Project{projectID: project},
	}
	uc1 := NewProjectUsecase(mockProjRepo, &mockRBACRepoForProjectUC{}, &mockUserRepoForProjectUC{})
	err := uc1.DeleteProject(ctx, notOwnerID, projectID)
	if !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("expected ErrForbidden for non-owner delete, got %v", err)
	}

	// 2. UpdateProject repo error
	uc2 := NewProjectUsecase(&failProjectRepo{ProjectRepository: mockProjRepo, err: dbErr}, &mockRBACRepoForProjectUC{}, &mockUserRepoForProjectUC{})
	_, err = uc2.UpdateProject(ctx, ownerID, projectID, domain.UpdateProjectRequest{Name: "New"})
	if err == nil {
		t.Error("expected error on UpdateProject repo failure")
	}

	// 3. AddMember user not found
	uc3 := NewProjectUsecase(mockProjRepo, &mockRBACRepoForProjectUC{}, &failUserRepo{err: domain.ErrNotFound})
	_, err = uc3.AddMember(ctx, projectID, domain.AddMemberRequest{Email: "missing@test.com", RoleName: "ADMIN"})
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound when user missing, got %v", err)
	}

	// 4. AddMember role not found
	validUser := &domain.User{ID: uuid.New(), Email: "found@test.com"}
	userRepo := &mockUserRepoForProjectUC{users: map[string]*domain.User{"found@test.com": validUser}}
	uc4 := NewProjectUsecase(mockProjRepo, &failRBACRepo{err: domain.ErrNotFound}, userRepo)
	_, err = uc4.AddMember(ctx, projectID, domain.AddMemberRequest{Email: "found@test.com", RoleName: "ADMIN"})
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound when role missing, got %v", err)
	}

	// 5. UpdateMemberRole role not found
	uc5 := NewProjectUsecase(mockProjRepo, &failRBACRepo{err: domain.ErrNotFound}, userRepo)
	_, err = uc5.UpdateMemberRole(ctx, projectID, validUser.ID, domain.UpdateMemberRoleRequest{RoleName: "INVALID"})
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound for invalid role name, got %v", err)
	}

	// 6. RemoveMember cannot remove project owner -> ErrForbidden
	uc6 := NewProjectUsecase(mockProjRepo, &mockRBACRepoForProjectUC{}, userRepo)
	err = uc6.RemoveMember(ctx, projectID, ownerID)
	if !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("expected ErrForbidden when removing owner, got %v", err)
	}
}

func TestMetricUsecase_ExhaustiveErrors(t *testing.T) {
	ctx := context.Background()
	projectID := uuid.New()
	sprintID := uuid.New()
	otherProjectID := uuid.New()
	dbErr := errors.New("db error")

	sprintInOtherProj := &domain.Sprint{
		ID:        sprintID,
		ProjectID: otherProjectID,
	}
	sprintRepo := &mockSprintRepoForSprintUC{
		sprints: map[uuid.UUID]*domain.Sprint{sprintID: sprintInOtherProj},
	}
	mockProjRepo := &mockProjectRepoForProjectUC{
		projectsByID: map[uuid.UUID]*domain.Project{projectID: {ID: projectID}},
	}

	// 1. GetSprintMetrics with sprint belonging to different project -> ErrNotFound
	uc1 := NewMetricUsecase(&failMetricRepo{}, &mockIssueRepoForSprintUC{}, mockProjRepo, sprintRepo)
	_, err := uc1.GetSprintMetrics(ctx, projectID, sprintID)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound for sprint in another project, got %v", err)
	}

	// 2. ExportIssuesCSV list error
	uc2 := NewMetricUsecase(&failMetricRepo{}, &failIssueRepo{err: dbErr}, mockProjRepo, sprintRepo)
	buf := new(bytes.Buffer)
	err = uc2.ExportIssuesCSV(ctx, domain.IssueFilter{ProjectID: projectID}, buf)
	if err == nil {
		t.Error("expected error on CSV export list failure")
	}

	// 3. ExportProjectSummary metrics failure
	uc3 := NewMetricUsecase(&failMetricRepo{err: dbErr}, &mockIssueRepoForSprintUC{}, mockProjRepo, sprintRepo)
	_, err = uc3.ExportProjectSummary(ctx, projectID)
	if err == nil {
		t.Error("expected error on ExportProjectSummary metrics failure")
	}
}
