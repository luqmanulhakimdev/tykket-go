package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/luqmanulhakimdev/tykket-go/internal/config"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("failed to load configuration: %v", err)
	}

	// pgx5 driver URL format for golang-migrate
	dsn := fmt.Sprintf("pgx5://%s:%s@%s:%s/%s?sslmode=%s",
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.Name,
		cfg.Database.SSLMode,
	)

	migrationsPath := "file://migrations"

	m, err := migrate.New(migrationsPath, dsn)
	if err != nil {
		log.Fatalf("failed to initialize migrate: %v", err)
	}
	defer func() {
		sourceErr, dbErr := m.Close()
		if sourceErr != nil {
			log.Printf("source close error: %v", sourceErr)
		}
		if dbErr != nil {
			log.Printf("db close error: %v", dbErr)
		}
	}()

	switch command {
	case "up":
		if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			log.Fatalf("migrate up failed: %v", err)
		}
		log.Println("Database migration UP applied successfully.")

	case "down":
		if err := m.Down(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			log.Fatalf("migrate down failed: %v", err)
		}
		log.Println("Database migration DOWN applied successfully.")

	case "step":
		stepCmd := flag.NewFlagSet("step", flag.ExitOnError)
		n := stepCmd.Int("n", 1, "number of steps (can be negative)")
		_ = stepCmd.Parse(os.Args[2:])
		if err := m.Steps(*n); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			log.Fatalf("migrate step failed: %v", err)
		}
		log.Printf("Database migration step %d applied successfully.\n", *n)

	case "force":
		if len(os.Args) < 3 {
			log.Fatalf("force requires version argument: migrate force <version>")
		}
		var version int
		if _, err := fmt.Sscanf(os.Args[2], "%d", &version); err != nil {
			log.Fatalf("invalid version: %v", err)
		}
		if err := m.Force(version); err != nil {
			log.Fatalf("migrate force failed: %v", err)
		}
		log.Printf("Database migration forced to version %d.\n", version)

	case "version":
		version, dirty, err := m.Version()
		if err != nil {
			if errors.Is(err, migrate.ErrNilVersion) {
				log.Println("No migration version applied yet.")
				return
			}
			log.Fatalf("failed to get migration version: %v", err)
		}
		log.Printf("Current migration version: %d (dirty: %v)\n", version, dirty)

	case "create":
		if len(os.Args) < 3 {
			log.Fatalf("create requires name argument: migrate create <name>")
		}
		name := os.Args[2]
		timestamp := time.Now().Unix()
		upFile := fmt.Sprintf("migrations/%06d_%s.up.sql", timestamp, name)
		downFile := fmt.Sprintf("migrations/%06d_%s.down.sql", timestamp, name)

		if err := os.WriteFile(upFile, []byte("-- Write your UP migration here\n"), 0644); err != nil {
			log.Fatalf("failed to create %s: %v", upFile, err)
		}
		if err := os.WriteFile(downFile, []byte("-- Write your DOWN migration here\n"), 0644); err != nil {
			log.Fatalf("failed to create %s: %v", downFile, err)
		}
		log.Printf("Created migration files:\n  %s\n  %s\n", upFile, downFile)

	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage: go run cmd/migrate/main.go <command> [arguments]")
	fmt.Println("Commands:")
	fmt.Println("  up                Apply all up migrations")
	fmt.Println("  down              Roll back all down migrations")
	fmt.Println("  step -n <count>   Apply +/- step migrations")
	fmt.Println("  force <version>   Force set migration version")
	fmt.Println("  version           Show current migration version")
	fmt.Println("  create <name>     Create new up/down migration files")
}

