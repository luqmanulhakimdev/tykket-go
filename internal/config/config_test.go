package config

import (
	"os"
	"testing"
)

func TestConfig(t *testing.T) {
	t.Run("LoadConfig defaults", func(t *testing.T) {
		cfg, err := LoadConfig()
		if err != nil {
			t.Fatalf("unexpected error loading config: %v", err)
		}

		if cfg.App.Port == "" {
			t.Errorf("expected default port, got empty")
		}

		dsn := cfg.Database.DSN()
		if dsn == "" {
			t.Errorf("expected non-empty DSN")
		}
	})

	t.Run("CORS Origins variations", func(t *testing.T) {
		c1 := CORSConfig{AllowedOrigins: "*"}
		if len(c1.Origins()) != 1 || c1.Origins()[0] != "*" {
			t.Errorf("expected ['*'], got %v", c1.Origins())
		}

		c2 := CORSConfig{AllowedOrigins: ""}
		if len(c2.Origins()) != 1 || c2.Origins()[0] != "*" {
			t.Errorf("expected ['*'], got %v", c2.Origins())
		}

		c3 := CORSConfig{AllowedOrigins: "http://localhost:3000, http://localhost:5173, "}
		origins := c3.Origins()
		if len(origins) != 2 {
			t.Errorf("expected 2 trimmed origins, got %d (%v)", len(origins), origins)
		}
		if origins[0] != "http://localhost:3000" || origins[1] != "http://localhost:5173" {
			t.Errorf("unexpected origins parsed: %v", origins)
		}
	})

	t.Run("Environment overrides", func(t *testing.T) {
		os.Setenv("APP_PORT", "9999")
		defer os.Unsetenv("APP_PORT")

		cfg, err := LoadConfig()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.App.Port != "9999" {
			t.Errorf("expected overridden port 9999, got %s", cfg.App.Port)
		}
	})
}
