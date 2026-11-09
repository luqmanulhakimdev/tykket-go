package postgres

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/luqmanulhakimdev/tykket-go/internal/domain"
)

func strPtr(s string) *string { return &s }

func TestProjectRepository_ConcurrentAtomicCounter(t *testing.T) {
	pool, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	userRepo := NewUserRepository(pool)
	projectRepo := NewProjectRepository(pool)
	issueRepo := NewIssueRepository(pool)

	// Create test user
	user := &domain.User{
		ID:           uuid.New(),
		Email:        fmt.Sprintf("concurrent_%d@tykket.dev", time.Now().UnixNano()),
		Name:         "Concurrent Tester",
		AuthProvider: domain.AuthProviderLocal,
	}
	if err := userRepo.Create(ctx, user); err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	// Create test project
	projectKey := fmt.Sprintf("C%d", time.Now().UnixNano()%100000)
	project := &domain.Project{
		ID:          uuid.New(),
		Name:        "Concurrent Test Project",
		Key:         projectKey,
		Description: strPtr("Testing atomic issue numbering under concurrency"),
		OwnerID:     user.ID,
	}
	if err := projectRepo.Create(ctx, project); err != nil {
		t.Fatalf("failed to create project: %v", err)
	}
	defer func() {
		_ = projectRepo.Delete(ctx, project.ID)
	}()

	// 1. Concurrent IncrementIssueCounter simulation
	const numWorkers = 50
	var wg sync.WaitGroup
	var mu sync.Mutex
	counters := make([]int64, 0, numWorkers)
	errs := make([]error, 0, numWorkers)

	wg.Add(numWorkers)
	for i := 0; i < numWorkers; i++ {
		go func() {
			defer wg.Done()
			key, c, err := projectRepo.IncrementIssueCounter(ctx, project.ID)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				errs = append(errs, err)
				return
			}
			if key != projectKey {
				errs = append(errs, fmt.Errorf("expected key %s, got %s", projectKey, key))
				return
			}
			counters = append(counters, c)
		}()
	}
	wg.Wait()

	if len(errs) > 0 {
		t.Fatalf("encountered %d errors during concurrent increments: %v", len(errs), errs[0])
	}

	if len(counters) != numWorkers {
		t.Fatalf("expected %d counter values, got %d", numWorkers, len(counters))
	}

	// Verify uniqueness
	uniqueCheck := make(map[int64]bool)
	for _, c := range counters {
		if uniqueCheck[c] {
			t.Fatalf("collision detected! Duplicate issue number: %d", c)
		}
		uniqueCheck[c] = true
	}

	// Verify all numbers from 1 to numWorkers are present
	for i := int64(1); i <= int64(numWorkers); i++ {
		if !uniqueCheck[i] {
			t.Fatalf("missing counter value: %d", i)
		}
	}

	// 2. Concurrent Issue Insertions with Unique Keys
	const numIssues = 20
	var issueWg sync.WaitGroup
	createdKeys := make(map[string]bool)
	var keyMu sync.Mutex

	issueWg.Add(numIssues)
	for i := 0; i < numIssues; i++ {
		go func(idx int) {
			defer issueWg.Done()
			_, counterVal, err := projectRepo.IncrementIssueCounter(ctx, project.ID)
			if err != nil {
				t.Errorf("failed to increment counter: %v", err)
				return
			}

			issueKey := fmt.Sprintf("%s-%d", projectKey, counterVal)
			newIssue := &domain.Issue{
				ID:          uuid.New(),
				ProjectID:   project.ID,
				Key:         issueKey,
				IssueNumber: counterVal,
				Type:        domain.IssueTypeTask,
				Status:      domain.IssueStatusTodo,
				Priority:    domain.IssuePriorityMedium,
				Title:       fmt.Sprintf("Concurrent Task %d", counterVal),
				ReporterID:  user.ID,
				Position:    float64(counterVal) * 1000.0,
			}

			if err := issueRepo.Create(ctx, newIssue); err != nil {
				t.Errorf("failed to insert issue concurrently: %v", err)
				return
			}

			keyMu.Lock()
			createdKeys[issueKey] = true
			keyMu.Unlock()
		}(i)
	}
	issueWg.Wait()

	if len(createdKeys) != numIssues {
		t.Fatalf("expected %d uniquely inserted issues, got %d", numIssues, len(createdKeys))
	}
}
