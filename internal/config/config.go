package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	App      AppConfig      `mapstructure:",squash"`
	Database DatabaseConfig `mapstructure:",squash"`
	JWT      JWTConfig      `mapstructure:",squash"`
	OAuth    OAuthConfig    `mapstructure:",squash"`
	CORS     CORSConfig     `mapstructure:",squash"`
}

type AppConfig struct {
	Env  string `mapstructure:"APP_ENV"`
	Port string `mapstructure:"APP_PORT"`
	Name string `mapstructure:"APP_NAME"`
}

type DatabaseConfig struct {
	Host            string        `mapstructure:"DB_HOST"`
	Port            string        `mapstructure:"DB_PORT"`
	User            string        `mapstructure:"DB_USER"`
	Password        string        `mapstructure:"DB_PASSWORD"`
	Name            string        `mapstructure:"DB_NAME"`
	SSLMode         string        `mapstructure:"DB_SSLMODE"`
	MaxConns        int32         `mapstructure:"DB_MAX_CONNS"`
	MinConns        int32         `mapstructure:"DB_MIN_CONNS"`
	MaxConnLifetime time.Duration `mapstructure:"DB_MAX_CONN_LIFETIME"`
	MaxConnIdleTime time.Duration `mapstructure:"DB_MAX_CONN_IDLE_TIME"`
}

func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		d.User, d.Password, d.Host, d.Port, d.Name, d.SSLMode)
}

type JWTConfig struct {
	Secret        string        `mapstructure:"JWT_SECRET"`
	AccessExpiry  time.Duration `mapstructure:"JWT_ACCESS_EXPIRY"`
	RefreshExpiry time.Duration `mapstructure:"JWT_REFRESH_EXPIRY"`
}

type OAuthConfig struct {
	GoogleClientID     string `mapstructure:"GOOGLE_CLIENT_ID"`
	GoogleClientSecret string `mapstructure:"GOOGLE_CLIENT_SECRET"`
	GoogleRedirectURL  string `mapstructure:"GOOGLE_REDIRECT_URL"`
}

type CORSConfig struct {
	AllowedOrigins string `mapstructure:"CORS_ALLOWED_ORIGINS"`
}

func (c CORSConfig) Origins() []string {
	if c.AllowedOrigins == "" || c.AllowedOrigins == "*" {
		return []string{"*"}
	}
	parts := strings.Split(c.AllowedOrigins, ",")
	res := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			res = append(res, trimmed)
		}
	}
	return res
}

func LoadConfig() (*Config, error) {
	v := viper.New()

	v.SetDefault("APP_ENV", "development")
	v.SetDefault("APP_PORT", "8080")
	v.SetDefault("APP_NAME", "tykket-api")

	v.SetDefault("DB_HOST", "localhost")
	v.SetDefault("DB_PORT", "5432")
	v.SetDefault("DB_USER", "postgres")
	v.SetDefault("DB_PASSWORD", "postgres")
	v.SetDefault("DB_NAME", "tykket_db")
	v.SetDefault("DB_SSLMODE", "disable")
	v.SetDefault("DB_MAX_CONNS", 25)
	v.SetDefault("DB_MIN_CONNS", 5)
	v.SetDefault("DB_MAX_CONN_LIFETIME", time.Hour)
	v.SetDefault("DB_MAX_CONN_IDLE_TIME", 30*time.Minute)

	v.SetDefault("JWT_SECRET", "default-insecure-secret-key-32-chars-long")
	v.SetDefault("JWT_ACCESS_EXPIRY", 15*time.Minute)
	v.SetDefault("JWT_REFRESH_EXPIRY", 7*24*time.Hour)

	v.SetDefault("CORS_ALLOWED_ORIGINS", "*")

	// Read from .env if present
	v.SetConfigFile(".env")
	v.SetConfigType("env")
	_ = v.ReadInConfig()

	// Automatic env overriding
	v.AutomaticEnv()

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unable to decode config: %w", err)
	}

	return &cfg, nil
}

