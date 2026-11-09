package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/luqmanulhakimdev/tykket-go/internal/config"
	"github.com/luqmanulhakimdev/tykket-go/internal/domain"
	"github.com/luqmanulhakimdev/tykket-go/pkg/database"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("failed to load configuration: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := database.NewPostgresPool(ctx, cfg.Database)
	if err != nil {
		log.Fatalf("failed to connect to postgres: %v", err)
	}
	defer pool.Close()

	log.Println("Seeding database with demo data for Frontend (Phase 2)...")

	if err := seedData(ctx, pool); err != nil {
		log.Fatalf("seeding failed: %v", err)
	}

	log.Println("Database successfully seeded!")
	log.Println("Demo Accounts (Password: password123):")
	log.Println("  - owner@tykket.dev   (Owner)")
	log.Println("  - admin@tykket.dev   (Admin)")
	log.Println("  - member@tykket.dev  (Member)")
	log.Println("  - viewer@tykket.dev  (Viewer)")
}

func seedData(ctx context.Context, pool *pgxpool.Pool) error {
	// Clean existing data for idempotency
	_, err := pool.Exec(ctx, `
		TRUNCATE TABLE issues, sprints, project_members, projects, users CASCADE;
	`)
	if err != nil {
		return fmt.Errorf("failed to truncate existing tables: %w", err)
	}

	// 1. Fetch Role IDs
	var (
		ownerRoleID  uuid.UUID
		adminRoleID  uuid.UUID
		memberRoleID uuid.UUID
		viewerRoleID uuid.UUID
	)
	_ = pool.QueryRow(ctx, "SELECT id FROM roles WHERE name = 'OWNER'").Scan(&ownerRoleID)
	_ = pool.QueryRow(ctx, "SELECT id FROM roles WHERE name = 'ADMIN'").Scan(&adminRoleID)
	_ = pool.QueryRow(ctx, "SELECT id FROM roles WHERE name = 'MEMBER'").Scan(&memberRoleID)
	_ = pool.QueryRow(ctx, "SELECT id FROM roles WHERE name = 'VIEWER'").Scan(&viewerRoleID)

	// 2. Hash Password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	pwdStr := string(hashedPassword)

	// 3. Create Users
	ownerID := uuid.New()
	adminID := uuid.New()
	memberID := uuid.New()
	viewerID := uuid.New()

	users := []struct {
		id    uuid.UUID
		name  string
		email string
	}{
		{ownerID, "Alex Mercer (Owner)", "owner@tykket.dev"},
		{adminID, "Sarah Connor (Admin)", "admin@tykket.dev"},
		{memberID, "David Kim (Engineer)", "member@tykket.dev"},
		{viewerID, "Emma Watson (Stakeholder)", "viewer@tykket.dev"},
	}

	for _, u := range users {
		_, err := pool.Exec(ctx, `
			INSERT INTO users (id, name, email, password_hash, auth_provider, created_at, updated_at)
			VALUES ($1, $2, $3, $4, 'LOCAL', NOW(), NOW())
		`, u.id, u.name, u.email, pwdStr)
		if err != nil {
			return fmt.Errorf("failed to insert user %s: %w", u.email, err)
		}
	}

	// 4. Create Project
	projectID := uuid.New()
	_, err = pool.Exec(ctx, `
		INSERT INTO projects (id, name, key, description, owner_id, issue_counter, created_at, updated_at)
		VALUES ($1, 'Tykket Core Platform', 'TYK', 'Main workspace for Tykket core frontend & backend development', $2, 11, NOW(), NOW())
	`, projectID, ownerID)
	if err != nil {
		return fmt.Errorf("failed to insert project: %w", err)
	}

	// 5. Add Members
	members := []struct {
		userID uuid.UUID
		roleID uuid.UUID
	}{
		{ownerID, ownerRoleID},
		{adminID, adminRoleID},
		{memberID, memberRoleID},
		{viewerID, viewerRoleID},
	}
	for _, m := range members {
		_, err := pool.Exec(ctx, `
			INSERT INTO project_members (id, project_id, user_id, role_id, joined_at)
			VALUES ($1, $2, $3, $4, NOW())
		`, uuid.New(), projectID, m.userID, m.roleID)
		if err != nil {
			return fmt.Errorf("failed to insert member: %w", err)
		}
	}

	// 6. Create Sprints
	now := time.Now().UTC()
	sprint1Start := now.Add(-14 * 24 * time.Hour)
	sprint1End := now.Add(-1 * 24 * time.Hour)
	sprint2Start := now.Add(-1 * 24 * time.Hour)
	sprint2End := now.Add(13 * 24 * time.Hour)

	sprint1ID := uuid.New()
	sprint2ID := uuid.New()
	sprint3ID := uuid.New()

	_, _ = pool.Exec(ctx, `
		INSERT INTO sprints (id, project_id, name, goal, start_date, end_date, status, created_at, updated_at)
		VALUES 
		($1, $4, 'Sprint 1 - Foundation & Auth', 'Setup project scaffolding and authentication', $5, $6, 'COMPLETED', NOW(), NOW()),
		($2, $4, 'Sprint 2 - Issue Board & Workflows', 'Implement Kanban board, fractional indexing and sprint lifecycle', $7, $8, 'ACTIVE', NOW(), NOW()),
		($3, $4, 'Sprint 3 - Analytics & Data Export', 'Reporting metrics, burndown charts and CSV export pipelines', NULL, NULL, 'PLANNED', NOW(), NOW())
	`, sprint1ID, sprint2ID, sprint3ID, projectID, sprint1Start, sprint1End, sprint2Start, sprint2End)

	// 7. Create Issues
	epic1ID := uuid.New()
	story1ID := uuid.New()
	task1ID := uuid.New()
	epic2ID := uuid.New()
	story2ID := uuid.New()
	subtask1ID := uuid.New()
	subtask2ID := uuid.New()
	task2ID := uuid.New()
	bug1ID := uuid.New()
	story3ID := uuid.New()
	task3ID := uuid.New()

	issues := []struct {
		id       uuid.UUID
		sprintID *uuid.UUID
		parentID *uuid.UUID
		key      string
		number   int64
		itype    domain.IssueType
		status   domain.IssueStatus
		priority domain.IssuePriority
		title    string
		desc     string
		assignee *uuid.UUID
		points   int
		position float64
	}{
		// Sprint 1 (Completed)
		{epic1ID, nil, nil, "TYK-1", 1, domain.IssueTypeEpic, domain.IssueStatusDone, domain.IssuePriorityHigh, "Core Architecture & Clean Architecture Scaffolding", "Setup domain, repo, usecase, delivery layers and PostgreSQL pgxpool", &ownerID, 0, 1000.0},
		{story1ID, &sprint1ID, &epic1ID, "TYK-2", 2, domain.IssueTypeStory, domain.IssueStatusDone, domain.IssuePriorityHigh, "User Registration & JWT Authentication Flow", "Implement bcrypt hashing, access token and refresh token rotation", &adminID, 5, 2000.0},
		{task1ID, &sprint1ID, &epic1ID, "TYK-3", 3, domain.IssueTypeTask, domain.IssueStatusDone, domain.IssuePriorityMedium, "Database Migration CLI & Makefile", "Setup golang-migrate integration with pgx5 driver", &memberID, 3, 3000.0},

		// Sprint 2 (Active Board)
		{epic2ID, nil, nil, "TYK-4", 4, domain.IssueTypeEpic, domain.IssueStatusInProgress, domain.IssuePriorityHigh, "Agile Board & Core Ticket Engine", "Epic tracking interactive ticket management and fractional indexing reorder", &ownerID, 0, 4000.0},
		{story2ID, &sprint2ID, &epic2ID, "TYK-5", 5, domain.IssueTypeStory, domain.IssueStatusInProgress, domain.IssuePriorityUrgent, "Interactive Kanban Board Movement with Fractional Indexing", "Support drag-and-drop between columns with instant O(1) order calculation", &memberID, 8, 10000.0},
		{subtask1ID, &sprint2ID, &story2ID, "TYK-6", 6, domain.IssueTypeSubtask, domain.IssueStatusDone, domain.IssuePriorityMedium, "Implement PATCH /issues/:id/move endpoint", "Handle position and status updates in single transaction", &memberID, 3, 10100.0},
		{subtask2ID, &sprint2ID, &story2ID, "TYK-7", 7, domain.IssueTypeSubtask, domain.IssueStatusInProgress, domain.IssuePriorityHigh, "Frontend Board Column Drop Handlers", "Calculate (prev + next) / 2 fractional index position on drop", &memberID, 5, 10200.0},
		{task2ID, &sprint2ID, nil, "TYK-8", 8, domain.IssueTypeTask, domain.IssueStatusTodo, domain.IssuePriorityHigh, "Sprint Burndown Chart Data Processing", "Compute ideal burndown versus actual remaining story points", &adminID, 5, 20000.0},
		{bug1ID, &sprint2ID, nil, "TYK-9", 9, domain.IssueTypeBug, domain.IssueStatusInReview, domain.IssuePriorityUrgent, "Fix CORS Header configuration for Frontend local dev port (3000)", "Allow Authorization and Content-Type headers explicitly in Echo CORS middleware", &memberID, 2, 30000.0},

		// Backlog
		{story3ID, nil, nil, "TYK-10", 10, domain.IssueTypeStory, domain.IssueStatusTodo, domain.IssuePriorityLow, "Data Export to CSV and Project Summary Report", "Stream ticket records as CSV and export project JSON summary", nil, 5, 40000.0},
		{task3ID, nil, nil, "TYK-11", 11, domain.IssueTypeTask, domain.IssueStatusTodo, domain.IssuePriorityMedium, "Google OAuth SSO Integration Flow", "Verify provider ID token and auto-link user accounts", nil, 8, 50000.0},
	}

	for _, it := range issues {
		_, err := pool.Exec(ctx, `
			INSERT INTO issues (
				id, project_id, sprint_id, parent_id, key, issue_number,
				type, status, priority, title, description,
				reporter_id, assignee_id, story_points, position,
				created_at, updated_at
			) VALUES (
				$1, $2, $3, $4, $5, $6,
				$7, $8, $9, $10, $11,
				$12, $13, $14, $15,
				NOW(), NOW()
			)
		`,
			it.id, projectID, it.sprintID, it.parentID, it.key, it.number,
			it.itype, it.status, it.priority, it.title, it.desc,
			ownerID, it.assignee, it.points, it.position,
		)
		if err != nil {
			return fmt.Errorf("failed to insert issue %s: %w", it.key, err)
		}
	}

	return nil
}

