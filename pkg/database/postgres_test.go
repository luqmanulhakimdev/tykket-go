package database

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/luqmanulhakimdev/tykket-go/internal/config"
)

func TestNewPostgresPool(t *testing.T) {
	if os.Getenv("DB_PORT") == "" {
		_ = os.Setenv("DB_PORT", "5434")
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	t.Run("Connect to valid postgres pool", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		pool, err := NewPostgresPool(ctx, cfg.Database)
		if err != nil {
			t.Skipf("skipping connection test: %v", err)
		}
		defer pool.Close()

		if err := pool.Ping(ctx); err != nil {
			t.Fatalf("failed to ping postgres pool: %v", err)
		}
	})

	t.Run("Fail on invalid host/port", func(t *testing.T) {
		badCfg := cfg.Database
		badCfg.Host = "127.0.0.1"
		badCfg.Port = "54321"

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		_, err := NewPostgresPool(ctx, badCfg)
		if err == nil {
			t.Fatal("expected error connecting to non-existent port, got nil")
		}
	})
}
