package postgres

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/luqmanulhakimdev/tykket-go/internal/config"
	"github.com/luqmanulhakimdev/tykket-go/internal/domain"
	"github.com/luqmanulhakimdev/tykket-go/pkg/database"
)

func setupTestDB(t *testing.T) (*pgxpool.Pool, func()) {
	// If DB_PORT is not set in env, use port 5434 where docker container runs
	if os.Getenv("DB_PORT") == "" {
		_ = os.Setenv("DB_PORT", "5434")
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Skipf("skipping db tests: failed to load config: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := database.NewPostgresPool(ctx, cfg.Database)
	if err != nil {
		t.Skipf("skipping db tests: postgres not accessible on %s:%s: %v", cfg.Database.Host, cfg.Database.Port, err)
	}

	teardown := func() {
		pool.Close()
	}

	return pool, teardown
}

func TestPostgresRepositories(t *testing.T) {
	pool, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	userRepo := NewUserRepository(pool)
	rbacRepo := NewRBACRepository(pool)
	projectRepo := NewProjectRepository(pool)
	sprintRepo := NewSprintRepository(pool)
	issueRepo := NewIssueRepository(pool)
	metricRepo := NewMetricRepository(pool)

	// Clean tables before running
	_, _ = pool.Exec(ctx, "TRUNCATE issues, sprints, project_members, projects, users CASCADE;")

	// 1. User Repository Tests
	var (
		userID = uuid.New()
		email  = fmt.Sprintf("testuser_%d@example.com", time.Now().UnixNano())
		name   = "Integration User"
		pwd    = "hashedpwd123"
	)

	t.Run("UserRepo Create, GetByID, GetByEmail, Update", func(t *testing.T) {
		user := &domain.User{
			ID:           userID,
			Email:        email,
			Name:         name,
			PasswordHash: &pwd,
			AuthProvider: domain.AuthProviderLocal,
		}
		if err := userRepo.Create(ctx, user); err != nil {
			t.Fatalf("failed to create user: %v", err)
		}

		// Conflict check
		if err := userRepo.Create(ctx, user); !errors.Is(err, domain.ErrConflict) {
			t.Errorf("expected ErrConflict, got %v", err)
		}

		// GetByID
		found, err := userRepo.GetByID(ctx, userID)
		if err != nil || found.Email != email {
			t.Fatalf("failed to get user by id: %v", err)
		}

		// GetByID not found
		if _, err := userRepo.GetByID(ctx, uuid.New()); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("expected ErrNotFound, got %v", err)
		}

		// GetByEmail
		foundEmail, err := userRepo.GetByEmail(ctx, email)
		if err != nil || foundEmail.ID != userID {
			t.Fatalf("failed to get user by email: %v", err)
		}

		// GetByEmail not found
		if _, err := userRepo.GetByEmail(ctx, "nonexistent@example.com"); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("expected ErrNotFound for missing email, got %v", err)
		}

		// Update
		user.Name = "Updated Name"
		if err := userRepo.Update(ctx, user); err != nil {
			t.Fatalf("failed to update user: %v", err)
		}

		// Update not found
		missingUser := &domain.User{ID: uuid.New(), Name: "Ghost"}
		if err := userRepo.Update(ctx, missingUser); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("expected ErrNotFound updating missing user, got %v", err)
		}
	})

	t.Run("UserRepo OAuth Provider", func(t *testing.T) {
		providerID := "google-12345"
		oauthUser := &domain.User{
			ID:           uuid.New(),
			Email:        fmt.Sprintf("oauth_%d@example.com", time.Now().UnixNano()),
			Name:         "OAuth User",
			AuthProvider: domain.AuthProviderGoogle,
			ProviderID:   &providerID,
		}
		if err := userRepo.Create(ctx, oauthUser); err != nil {
			t.Fatalf("failed to create oauth user: %v", err)
		}

		found, err := userRepo.GetByProvider(ctx, domain.AuthProviderGoogle, providerID)
		if err != nil || found.ID != oauthUser.ID {
			t.Fatalf("failed to get user by provider: %v", err)
		}

		if _, err := userRepo.GetByProvider(ctx, domain.AuthProviderGoogle, "nonexistent"); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("expected ErrNotFound for wrong provider id, got %v", err)
		}
	})

	// 2. RBAC Repository Tests
	var (
		ownerRole *domain.Role
		adminRole *domain.Role
	)

	t.Run("RBACRepo ListRoles, GetRoleByName, GetRoleByID", func(t *testing.T) {
		roles, err := rbacRepo.ListRoles(ctx)
		if err != nil || len(roles) == 0 {
			t.Fatalf("failed to list roles: %v", err)
		}

		ownerRole, err = rbacRepo.GetRoleByName(ctx, "OWNER")
		if err != nil || ownerRole.Name != "OWNER" {
			t.Fatalf("failed to get role by name: %v", err)
		}

		if _, err := rbacRepo.GetRoleByName(ctx, "NON_EXISTENT"); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("expected ErrNotFound, got %v", err)
		}

		adminRole, err = rbacRepo.GetRoleByID(ctx, ownerRole.ID)
		if err != nil || adminRole.ID != ownerRole.ID {
			t.Fatalf("failed to get role by id: %v", err)
		}

		if _, err := rbacRepo.GetRoleByID(ctx, uuid.New()); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("expected ErrNotFound, got %v", err)
		}
	})

	// 3. Project Repository Tests
	var projectID = uuid.New()
	var projectKey = "TST"

	t.Run("ProjectRepo Create, GetByID, GetByKey, IncrementCounter, ListByUserID, Update, Delete", func(t *testing.T) {
		p := &domain.Project{
			ID:          projectID,
			Name:        "Test Integration Project",
			Key:         projectKey,
			Description: nil,
			OwnerID:     userID,
		}
		if err := projectRepo.Create(ctx, p); err != nil {
			t.Fatalf("failed to create project: %v", err)
		}

		// Conflict
		if err := projectRepo.Create(ctx, p); !errors.Is(err, domain.ErrConflict) {
			t.Errorf("expected ErrConflict on duplicate key, got %v", err)
		}

		// GetByID
		found, err := projectRepo.GetByID(ctx, projectID)
		if err != nil || found.Key != projectKey {
			t.Fatalf("failed to get project by id: %v", err)
		}

		if _, err := projectRepo.GetByID(ctx, uuid.New()); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("expected ErrNotFound for missing project, got %v", err)
		}

		// GetByKey
		foundKey, err := projectRepo.GetByKey(ctx, projectKey)
		if err != nil || foundKey.ID != projectID {
			t.Fatalf("failed to get project by key: %v", err)
		}

		if _, err := projectRepo.GetByKey(ctx, "NONEXIST"); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("expected ErrNotFound for missing key, got %v", err)
		}

		// IncrementIssueCounter
		key, counter, err := projectRepo.IncrementIssueCounter(ctx, projectID)
		if err != nil || key != projectKey || counter != 1 {
			t.Fatalf("unexpected issue counter result: key=%s, counter=%d, err=%v", key, counter, err)
		}

		if _, _, err := projectRepo.IncrementIssueCounter(ctx, uuid.New()); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("expected ErrNotFound incrementing missing project, got %v", err)
		}

		// ListByUserID
		list, err := projectRepo.ListByUserID(ctx, userID)
		if err != nil || len(list) == 0 {
			t.Fatalf("failed to list projects: %v", err)
		}

		// Update
		p.Name = "Updated Project Name"
		if err := projectRepo.Update(ctx, p); err != nil {
			t.Fatalf("failed to update project: %v", err)
		}

		missingP := &domain.Project{ID: uuid.New(), Name: "Ghost"}
		if err := projectRepo.Update(ctx, missingP); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("expected ErrNotFound updating missing project, got %v", err)
		}
	})

	// 4. RBAC Membership Tests
	t.Run("RBACRepo Member Management & HasPermission", func(t *testing.T) {
		memberID := uuid.New()
		pm := &domain.ProjectMember{
			ID:        memberID,
			ProjectID: projectID,
			UserID:    userID,
			RoleID:    ownerRole.ID,
		}
		if err := rbacRepo.AddProjectMember(ctx, pm); err != nil {
			t.Fatalf("failed to add member: %v", err)
		}

		// Conflict
		if err := rbacRepo.AddProjectMember(ctx, pm); !errors.Is(err, domain.ErrConflict) {
			t.Errorf("expected ErrConflict adding duplicate member, got %v", err)
		}

		// HasPermission
		hasPerm, err := rbacRepo.HasPermission(ctx, userID, projectID, domain.PermIssueCreate)
		if err != nil || !hasPerm {
			t.Fatalf("expected hasPerm true, got %v (err: %v)", hasPerm, err)
		}

		// GetUserMemberInProject
		m, err := rbacRepo.GetUserMemberInProject(ctx, userID, projectID)
		if err != nil || m.RoleID != ownerRole.ID {
			t.Fatalf("failed to get member: %v", err)
		}

		if _, err := rbacRepo.GetUserMemberInProject(ctx, uuid.New(), projectID); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("expected ErrNotFound for missing member, got %v", err)
		}

		// ListProjectMembers
		members, err := rbacRepo.ListProjectMembers(ctx, projectID)
		if err != nil || len(members) == 0 {
			t.Fatalf("failed to list project members: %v", err)
		}

		// Update role
		adminRoleObj, err := rbacRepo.GetRoleByName(ctx, "ADMIN")
		if err != nil {
			t.Fatalf("failed to get admin role: %v", err)
		}
		if err := rbacRepo.UpdateProjectMemberRole(ctx, projectID, userID, adminRoleObj.ID); err != nil {
			t.Fatalf("failed to update member role: %v", err)
		}

		if err := rbacRepo.UpdateProjectMemberRole(ctx, projectID, uuid.New(), adminRoleObj.ID); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("expected ErrNotFound updating missing member role, got %v", err)
		}

		// Remove member
		tempUser := &domain.User{
			ID:           uuid.New(),
			Email:        fmt.Sprintf("temp_%d@example.com", time.Now().UnixNano()),
			Name:         "Temp",
			AuthProvider: domain.AuthProviderLocal,
		}
		if err := userRepo.Create(ctx, tempUser); err != nil {
			t.Fatalf("failed to create temp user: %v", err)
		}
		if err := rbacRepo.AddProjectMember(ctx, &domain.ProjectMember{
			ID:        uuid.New(),
			ProjectID: projectID,
			UserID:    tempUser.ID,
			RoleID:    adminRoleObj.ID,
		}); err != nil {
			t.Fatalf("failed to add temp member: %v", err)
		}

		if err := rbacRepo.RemoveProjectMember(ctx, projectID, tempUser.ID); err != nil {
			t.Fatalf("failed to remove member: %v", err)
		}

		// Verify member is soft deleted
		if _, err := rbacRepo.GetUserMemberInProject(ctx, tempUser.ID, projectID); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("expected ErrNotFound for soft-deleted member, got %v", err)
		}
		permCheck, err := rbacRepo.HasPermission(ctx, tempUser.ID, projectID, domain.PermIssueCreate)
		if err != nil || permCheck {
			t.Errorf("expected false permission for soft-deleted member, got %v (err: %v)", permCheck, err)
		}
		var memberDelAt *time.Time
		if err := pool.QueryRow(ctx, "SELECT deleted_at FROM project_members WHERE project_id = $1 AND user_id = $2", projectID, tempUser.ID).Scan(&memberDelAt); err != nil || memberDelAt == nil {
			t.Errorf("expected non-nil deleted_at in DB for soft-deleted member, err: %v", err)
		}

		// Verify re-adding the member works due to partial unique index
		if err := rbacRepo.AddProjectMember(ctx, &domain.ProjectMember{
			ID:        uuid.New(),
			ProjectID: projectID,
			UserID:    tempUser.ID,
			RoleID:    adminRoleObj.ID,
		}); err != nil {
			t.Fatalf("failed to re-add soft-deleted member: %v", err)
		}
		// Clean up re-added member
		_ = rbacRepo.RemoveProjectMember(ctx, projectID, tempUser.ID)

		if err := rbacRepo.RemoveProjectMember(ctx, projectID, uuid.New()); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("expected ErrNotFound removing non-existent member, got %v", err)
		}
	})

	// 5. Sprint Repository Tests
	var sprintID = uuid.New()
	t.Run("SprintRepo CRUD & Active Sprint", func(t *testing.T) {
		s := &domain.Sprint{
			ID:        sprintID,
			ProjectID: projectID,
			Name:      "Integration Sprint 1",
			Status:    domain.SprintStatusPlanned,
		}
		if err := sprintRepo.Create(ctx, s); err != nil {
			t.Fatalf("failed to create sprint: %v", err)
		}

		// GetByID
		found, err := sprintRepo.GetByID(ctx, sprintID)
		if err != nil || found.Name != "Integration Sprint 1" {
			t.Fatalf("failed to get sprint by id: %v", err)
		}

		if _, err := sprintRepo.GetByID(ctx, uuid.New()); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("expected ErrNotFound for missing sprint, got %v", err)
		}

		// GetActiveByProjectID when none active -> nil, nil
		active, err := sprintRepo.GetActiveByProjectID(ctx, projectID)
		if err != nil || active != nil {
			t.Errorf("expected nil active sprint, got %v", active)
		}

		// Update to ACTIVE
		s.Status = domain.SprintStatusActive
		if err := sprintRepo.Update(ctx, s); err != nil {
			t.Fatalf("failed to update sprint: %v", err)
		}

		activeFound, err := sprintRepo.GetActiveByProjectID(ctx, projectID)
		if err != nil || activeFound == nil || activeFound.ID != sprintID {
			t.Fatalf("expected active sprint found, got %v (err: %v)", activeFound, err)
		}

		// ListByProjectID
		list, err := sprintRepo.ListByProjectID(ctx, projectID)
		if err != nil || len(list) == 0 {
			t.Fatalf("failed to list sprints: %v", err)
		}
	})

	// 6. Issue Repository Tests
	var issueID = uuid.New()
	t.Run("IssueRepo CRUD, Board Movement, BulkMove & Hierarchy", func(t *testing.T) {
		points := 5
		issue := &domain.Issue{
			ID:          issueID,
			ProjectID:   projectID,
			SprintID:    &sprintID,
			Key:         "TST-1",
			IssueNumber: 1,
			Type:        domain.IssueTypeTask,
			Status:      domain.IssueStatusTodo,
			Priority:    domain.IssuePriorityHigh,
			Title:       "Test Issue Integration",
			ReporterID:  userID,
			StoryPoints: &points,
			Position:    65535.0,
		}
		if err := issueRepo.Create(ctx, issue); err != nil {
			t.Fatalf("failed to create issue: %v", err)
		}

		// Conflict
		if err := issueRepo.Create(ctx, issue); !errors.Is(err, domain.ErrConflict) {
			t.Errorf("expected ErrConflict on duplicate issue key, got %v", err)
		}

		// GetByID
		found, err := issueRepo.GetByID(ctx, issueID)
		if err != nil || found.Key != "TST-1" {
			t.Fatalf("failed to get issue by id: %v", err)
		}

		if _, err := issueRepo.GetByID(ctx, uuid.New()); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("expected ErrNotFound for missing issue, got %v", err)
		}

		// GetByKey
		foundKey, err := issueRepo.GetByKey(ctx, "TST-1")
		if err != nil || foundKey.ID != issueID {
			t.Fatalf("failed to get issue by key: %v", err)
		}

		if _, err := issueRepo.GetByKey(ctx, "TST-999"); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("expected ErrNotFound for missing issue key, got %v", err)
		}

		// List with filters and sorting
		inBacklog := false
		taskType := domain.IssueTypeTask
		todoStatus := domain.IssueStatusTodo
		list, total, err := issueRepo.List(ctx, domain.IssueFilter{
			ProjectID:  projectID,
			SprintID:   &sprintID,
			InBacklog:  &inBacklog,
			Type:       &taskType,
			Status:     &todoStatus,
			ReporterID: &userID,
			Search:     "Integration",
			SortBy:     "created_at",
			SortOrder:  "DESC",
			Limit:      10,
			Offset:     0,
		})
		if err != nil || total == 0 || len(list) == 0 {
			t.Fatalf("failed to list filtered issues: total=%d, err=%v", total, err)
		}

		// Sort by status, priority, updated_at
		_, _, _ = issueRepo.List(ctx, domain.IssueFilter{ProjectID: projectID, SortBy: "status", SortOrder: "ASC"})
		_, _, _ = issueRepo.List(ctx, domain.IssueFilter{ProjectID: projectID, SortBy: "priority", SortOrder: "DESC"})
		_, _, _ = issueRepo.List(ctx, domain.IssueFilter{ProjectID: projectID, SortBy: "updated_at", SortOrder: "DESC"})

		// Update
		issue.Title = "Renamed Issue"
		if err := issueRepo.Update(ctx, issue); err != nil {
			t.Fatalf("failed to update issue: %v", err)
		}

		missingIssue := &domain.Issue{
			ID:       uuid.New(),
			Type:     domain.IssueTypeTask,
			Priority: domain.IssuePriorityLow,
			Title:    "Ghost",
		}
		if err := issueRepo.Update(ctx, missingIssue); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("expected ErrNotFound updating missing issue, got %v", err)
		}

		// UpdatePositionAndStatus with non-nil sprintID
		inProg := domain.IssueStatusInProgress
		if err := issueRepo.UpdatePositionAndStatus(ctx, issueID, &inProg, 32768.0, &sprintID); err != nil {
			t.Fatalf("failed to update position and status: %v", err)
		}

		// Sprint update not found
		missingSprint := &domain.Sprint{ID: uuid.New(), Name: "Missing", Status: domain.SprintStatusPlanned}
		if err := sprintRepo.Update(ctx, missingSprint); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("expected ErrNotFound updating missing sprint, got %v", err)
		}

		if err := issueRepo.UpdatePositionAndStatus(ctx, uuid.New(), &inProg, 100.0, nil); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("expected ErrNotFound moving missing issue, got %v", err)
		}

		// BulkMoveIssues
		var nilSprint *uuid.UUID = nil
		if err := issueRepo.BulkMoveIssues(ctx, []uuid.UUID{issueID}, nilSprint); err != nil {
			t.Fatalf("failed to bulk move issues: %v", err)
		}

		// MoveIncompleteIssues
		if err := issueRepo.MoveIncompleteIssues(ctx, sprintID, nilSprint); err != nil {
			t.Fatalf("failed to move incomplete issues: %v", err)
		}
	})

	// 7. Metric Repository Tests
	t.Run("MetricRepo Sprint & Project Metrics with Burndown", func(t *testing.T) {
		start := time.Now().Add(-7 * 24 * time.Hour)
		end := time.Now().Add(7 * 24 * time.Hour)
		sprintWithDates := &domain.Sprint{
			ID:        uuid.New(),
			ProjectID: projectID,
			Name:      "Burndown Sprint",
			StartDate: &start,
			EndDate:   &end,
			Status:    domain.SprintStatusActive,
		}
		_ = sprintRepo.Create(ctx, sprintWithDates)

		points := 10
		_ = issueRepo.Create(ctx, &domain.Issue{
			ID:          uuid.New(),
			ProjectID:   projectID,
			SprintID:    &sprintWithDates.ID,
			Key:         "TST-BURNDOWN",
			IssueNumber: 99,
			Type:        domain.IssueTypeStory,
			Status:      domain.IssueStatusDone,
			Priority:    domain.IssuePriorityHigh,
			Title:       "Done Story",
			ReporterID:  userID,
			AssigneeID:  &userID,
			StoryPoints: &points,
			Position:    100.0,
		})

		sMetrics, err := metricRepo.GetSprintMetrics(ctx, sprintWithDates.ID)
		if err != nil {
			t.Fatalf("failed to get sprint metrics: %v", err)
		}
		if len(sMetrics.Burndown) == 0 {
			t.Errorf("expected burndown calculation with dates, got 0 points")
		}

		pMetrics, err := metricRepo.GetProjectMetrics(ctx, projectID)
		if err != nil {
			t.Fatalf("failed to get project metrics: %v", err)
		}
		if pMetrics.TotalIssues == 0 {
			t.Errorf("expected > 0 issues in project metrics")
		}
	})

	// 8. Default Fields, Move Incomplete, Bulk Move, and Filtered List
	t.Run("Defaults and Filtered Queries", func(t *testing.T) {
		// User with uuid.Nil
		userNil := &domain.User{
			Email:        fmt.Sprintf("nil_%d@example.com", time.Now().UnixNano()),
			Name:         "Nil User",
			AuthProvider: domain.AuthProviderLocal,
		}
		if err := userRepo.Create(ctx, userNil); err != nil {
			t.Fatalf("failed to create user with nil ID: %v", err)
		}
		if userNil.ID == uuid.Nil {
			t.Errorf("expected generated UUID for user")
		}

		// Project with uuid.Nil
		projNil := &domain.Project{
			Name:    "Nil Project",
			Key:     fmt.Sprintf("N%d", time.Now().UnixNano()%100000),
			OwnerID: userNil.ID,
		}
		if err := projectRepo.Create(ctx, projNil); err != nil {
			t.Fatalf("failed to create project with nil ID: %v", err)
		}
		if projNil.ID == uuid.Nil {
			t.Errorf("expected generated UUID for project")
		}

		// Sprint with uuid.Nil and empty status
		sprintNil := &domain.Sprint{
			ProjectID: projNil.ID,
			Name:      "Nil Sprint",
		}
		if err := sprintRepo.Create(ctx, sprintNil); err != nil {
			t.Fatalf("failed to create sprint with nil ID: %v", err)
		}
		if sprintNil.ID == uuid.Nil || sprintNil.Status != domain.SprintStatusPlanned {
			t.Errorf("expected generated UUID and PLANNED status for sprint")
		}

		// Issue with uuid.Nil, empty status, empty priority, position 0
		issueNil := &domain.Issue{
			ProjectID:   projNil.ID,
			SprintID:    &sprintNil.ID,
			Key:         fmt.Sprintf("%s-1", projNil.Key),
			IssueNumber: 1,
			Type:        domain.IssueTypeTask,
			Title:       "Task Defaults",
			ReporterID:  userNil.ID,
		}
		if err := issueRepo.Create(ctx, issueNil); err != nil {
			t.Fatalf("failed to create issue with nil defaults: %v", err)
		}
		if issueNil.ID == uuid.Nil || issueNil.Status != domain.IssueStatusTodo || issueNil.Priority != domain.IssuePriorityMedium || issueNil.Position == 0 {
			t.Errorf("expected defaults applied to issue, got %+v", issueNil)
		}

		// Subtask under issueNil
		subtask := &domain.Issue{
			ProjectID:   projNil.ID,
			SprintID:    &sprintNil.ID,
			ParentID:    &issueNil.ID,
			Key:         fmt.Sprintf("%s-2", projNil.Key),
			IssueNumber: 2,
			Type:        domain.IssueTypeSubtask,
			Status:      domain.IssueStatusTodo,
			Priority:    domain.IssuePriorityLow,
			Title:       "Subtask of issueNil",
			ReporterID:  userNil.ID,
			Position:    10.0,
		}
		if err := issueRepo.Create(ctx, subtask); err != nil {
			t.Fatalf("failed to create subtask: %v", err)
		}

		// GetByID for parent should load subtasks
		parentWithSub, err := issueRepo.GetByID(ctx, issueNil.ID)
		if err != nil || len(parentWithSub.Subtasks) == 0 {
			t.Errorf("expected parent issue to load subtasks, got err: %v, subtasks: %d", err, len(parentWithSub.Subtasks))
		}

		// GetByKey for parent
		byKey, err := issueRepo.GetByKey(ctx, issueNil.Key)
		if err != nil || byKey.ID != issueNil.ID {
			t.Errorf("expected issue by key, got err: %v", err)
		}

		// MoveIncompleteIssues
		if err := issueRepo.MoveIncompleteIssues(ctx, sprintNil.ID, nil); err != nil {
			t.Fatalf("failed to move incomplete issues: %v", err)
		}

		// BulkMoveIssues
		if err := issueRepo.BulkMoveIssues(ctx, []uuid.UUID{issueNil.ID, subtask.ID}, &sprintNil.ID); err != nil {
			t.Fatalf("failed to bulk move issues: %v", err)
		}

		// Filtered List queries
		searchStr := "Defaults"
		statusFilter := domain.IssueStatusTodo
		typeFilter := domain.IssueTypeTask
		listFiltered, total, err := issueRepo.List(ctx, domain.IssueFilter{
			ProjectID:  projNil.ID,
			SprintID:   &sprintNil.ID,
			ParentID:   nil,
			AssigneeID: nil,
			ReporterID: &userNil.ID,
			Status:     &statusFilter,
			Type:       &typeFilter,
			Search:     searchStr,
			Limit:      10,
			Offset:     0,
		})
		if err != nil || total == 0 || len(listFiltered) == 0 {
			t.Errorf("expected filtered list to return issueNil, err: %v, total: %d", err, total)
		}

		// Clean up projNil
		_ = projectRepo.Delete(ctx, projNil.ID)
	})

	// 9. Soft Delete & Cascades
	t.Run("Delete operations and soft-delete verification", func(t *testing.T) {
		// Subtask cascade test setup
		parentIssue := &domain.Issue{
			ProjectID:   projectID,
			Key:         "CASCADE-1",
			IssueNumber: 100,
			Type:        domain.IssueTypeStory,
			Title:       "Parent Story for Cascade",
			ReporterID:  userID,
		}
		if err := issueRepo.Create(ctx, parentIssue); err != nil {
			t.Fatalf("failed to create parent issue for cascade test: %v", err)
		}
		childSubtask := &domain.Issue{
			ProjectID:   projectID,
			ParentID:    &parentIssue.ID,
			Key:         "CASCADE-2",
			IssueNumber: 101,
			Type:        domain.IssueTypeSubtask,
			Title:       "Child Subtask for Cascade",
			ReporterID:  userID,
		}
		if err := issueRepo.Create(ctx, childSubtask); err != nil {
			t.Fatalf("failed to create child subtask for cascade test: %v", err)
		}

		// Delete parent issue
		if err := issueRepo.Delete(ctx, parentIssue.ID); err != nil {
			t.Fatalf("failed to soft delete parent issue: %v", err)
		}
		// Confirm parent is not found via repository
		if _, err := issueRepo.GetByID(ctx, parentIssue.ID); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("expected ErrNotFound for soft-deleted parent issue, got %v", err)
		}
		// Confirm child subtask was cascaded to soft-deleted
		if _, err := issueRepo.GetByID(ctx, childSubtask.ID); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("expected ErrNotFound for cascaded soft-deleted subtask, got %v", err)
		}
		// Confirm raw database records still exist with deleted_at IS NOT NULL
		var parentDelAt, childDelAt *time.Time
		if err := pool.QueryRow(ctx, "SELECT deleted_at FROM issues WHERE id = $1", parentIssue.ID).Scan(&parentDelAt); err != nil || parentDelAt == nil {
			t.Errorf("expected non-nil deleted_at in DB for parent issue, err: %v", err)
		}
		if err := pool.QueryRow(ctx, "SELECT deleted_at FROM issues WHERE id = $1", childSubtask.ID).Scan(&childDelAt); err != nil || childDelAt == nil {
			t.Errorf("expected non-nil deleted_at in DB for child subtask, err: %v", err)
		}

		if err := issueRepo.Delete(ctx, issueID); err != nil {
			t.Fatalf("failed to delete issue: %v", err)
		}
		if err := issueRepo.Delete(ctx, uuid.New()); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("expected ErrNotFound deleting missing issue, got %v", err)
		}

		// Sprint delete
		if err := sprintRepo.Delete(ctx, sprintID); err != nil {
			t.Fatalf("failed to delete sprint: %v", err)
		}
		if _, err := sprintRepo.GetByID(ctx, sprintID); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("expected ErrNotFound getting soft-deleted sprint, got %v", err)
		}
		if err := sprintRepo.Delete(ctx, uuid.New()); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("expected ErrNotFound deleting missing sprint, got %v", err)
		}
		var sprintDelAt *time.Time
		if err := pool.QueryRow(ctx, "SELECT deleted_at FROM sprints WHERE id = $1", sprintID).Scan(&sprintDelAt); err != nil || sprintDelAt == nil {
			t.Errorf("expected non-nil deleted_at in DB for sprint, err: %v", err)
		}

		// Project delete (cascades to remaining members, sprints, issues)
		if err := projectRepo.Delete(ctx, projectID); err != nil {
			t.Fatalf("failed to delete project: %v", err)
		}
		if _, err := projectRepo.GetByID(ctx, projectID); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("expected ErrNotFound getting soft-deleted project, got %v", err)
		}
		if err := projectRepo.Delete(ctx, uuid.New()); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("expected ErrNotFound deleting missing project, got %v", err)
		}
		var projDelAt *time.Time
		if err := pool.QueryRow(ctx, "SELECT deleted_at FROM projects WHERE id = $1", projectID).Scan(&projDelAt); err != nil || projDelAt == nil {
			t.Errorf("expected non-nil deleted_at in DB for project, err: %v", err)
		}

		// Verify that soft-deleted project is excluded from list
		projList, err := projectRepo.ListByUserID(ctx, userID)
		if err != nil {
			t.Fatalf("failed to list projects after delete: %v", err)
		}
		for _, pr := range projList {
			if pr.ID == projectID {
				t.Errorf("expected soft deleted project %v to be excluded from ListByUserID", projectID)
			}
		}

		// Verify project issues are excluded from List
		postDelIssues, count, err := issueRepo.List(ctx, domain.IssueFilter{ProjectID: projectID})
		if err != nil || count != 0 || len(postDelIssues) != 0 {
			t.Errorf("expected 0 active issues for soft-deleted project, got %d (err: %v)", count, err)
		}

		// Verify sprints are excluded from List
		postDelSprints, err := sprintRepo.ListByProjectID(ctx, projectID)
		if err != nil || len(postDelSprints) != 0 {
			t.Errorf("expected 0 active sprints for soft-deleted project, got %d (err: %v)", len(postDelSprints), err)
		}

		// Verify members are excluded from List
		postDelMembers, err := rbacRepo.ListProjectMembers(ctx, projectID)
		if err != nil || len(postDelMembers) != 0 {
			t.Errorf("expected 0 active members for soft-deleted project, got %d (err: %v)", len(postDelMembers), err)
		}

		// Verify mutation operations on soft deleted project fail with ErrNotFound
		if err := projectRepo.Update(ctx, &domain.Project{ID: projectID, Name: "Revive"}); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("expected ErrNotFound updating soft-deleted project, got %v", err)
		}
		if _, _, err := projectRepo.IncrementIssueCounter(ctx, projectID); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("expected ErrNotFound incrementing soft-deleted project counter, got %v", err)
		}
	})
}
