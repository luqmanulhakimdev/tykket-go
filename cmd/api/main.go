package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	echoMiddleware "github.com/labstack/echo/v4/middleware"
	"github.com/luqmanulhakimdev/tykket-go/internal/config"
	deliveryHttp "github.com/luqmanulhakimdev/tykket-go/internal/delivery/http"
	"github.com/luqmanulhakimdev/tykket-go/internal/delivery/http/handler"
	appMiddleware "github.com/luqmanulhakimdev/tykket-go/internal/delivery/http/middleware"
	"github.com/luqmanulhakimdev/tykket-go/internal/repository/postgres"
	"github.com/luqmanulhakimdev/tykket-go/internal/usecase"
	"github.com/luqmanulhakimdev/tykket-go/pkg/database"
	"github.com/luqmanulhakimdev/tykket-go/pkg/response"
	"github.com/luqmanulhakimdev/tykket-go/pkg/token"
	"github.com/luqmanulhakimdev/tykket-go/pkg/validator"
)

func main() {
	// 1. Initialize Logger
	logHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	logger := slog.New(logHandler)
	slog.SetDefault(logger)

	logger.Info("Starting Tykket Backend API...")

	// 2. Load Configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		logger.Error("Failed to load configuration", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// 3. Connect to Database Pool
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dbPool, err := database.NewPostgresPool(ctx, cfg.Database)
	if err != nil {
		logger.Warn("Failed to establish initial PostgreSQL connection (service will start but DB features will fail until DB is online)",
			slog.String("error", err.Error()),
		)
	} else {
		defer dbPool.Close()
		logger.Info("Connected to PostgreSQL successfully",
			slog.String("host", cfg.Database.Host),
			slog.String("dbname", cfg.Database.Name),
		)
	}

	// 4. Initialize Core Infrastructure & Token Manager
	tokenMgr := token.NewJWTManager(cfg.JWT.Secret, cfg.JWT.AccessExpiry, cfg.JWT.RefreshExpiry)

	// 5. Initialize Repositories (PostgreSQL)
	userRepo := postgres.NewUserRepository(dbPool)
	rbacRepo := postgres.NewRBACRepository(dbPool)
	projectRepo := postgres.NewProjectRepository(dbPool)
	sprintRepo := postgres.NewSprintRepository(dbPool)
	issueRepo := postgres.NewIssueRepository(dbPool)
	metricRepo := postgres.NewMetricRepository(dbPool)

	// 6. Initialize Business Logic (Usecases)
	authUsecase := usecase.NewAuthUsecase(userRepo, tokenMgr)
	projectUsecase := usecase.NewProjectUsecase(projectRepo, rbacRepo, userRepo)
	sprintUsecase := usecase.NewSprintUsecase(sprintRepo, issueRepo)
	issueUsecase := usecase.NewIssueUsecase(issueRepo, projectRepo, sprintRepo)
	metricUsecase := usecase.NewMetricUsecase(metricRepo, issueRepo, projectRepo, sprintRepo)

	// 7. Initialize Delivery Handlers
	authHandler := handler.NewAuthHandler(authUsecase)
	projectHandler := handler.NewProjectHandler(projectUsecase)
	sprintHandler := handler.NewSprintHandler(sprintUsecase)
	issueHandler := handler.NewIssueHandler(issueUsecase)
	metricHandler := handler.NewMetricHandler(metricUsecase)

	// 8. Initialize HTTP Server & Common Middleware
	e := echo.New()
	e.HideBanner = true
	e.Validator = validator.NewCustomValidator()

	// Global Middlewares
	e.Use(echoMiddleware.RequestID())
	e.Use(appMiddleware.StructuredLogger(logger))
	e.Use(appMiddleware.Recover(logger))
	e.Use(echoMiddleware.CORSWithConfig(echoMiddleware.CORSConfig{
		AllowOrigins: cfg.CORS.Origins(),
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization, echo.HeaderXRequestID},
	}))

	// Base Healthcheck route
	e.GET("/healthz", func(c echo.Context) error {
		dbStatus := "connected"
		if dbPool == nil {
			dbStatus = "disconnected"
		} else if err := dbPool.Ping(c.Request().Context()); err != nil {
			dbStatus = fmt.Sprintf("unhealthy: %v", err)
		}

		return response.OK(c, "Tykket API is alive", map[string]any{
			"status":   "UP",
			"env":      cfg.App.Env,
			"database": dbStatus,
			"time":     time.Now().UTC(),
		})
	})

	// Interactive Swagger UI & OpenAPI Specification
	e.File("/docs/openapi.yaml", "docs/openapi.yaml")
	e.GET("/docs", func(c echo.Context) error {
		html := `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>Tykket API Documentation</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css" />
</head>
<body>
<div id="swagger-ui"></div>
<script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
<script>
  window.onload = () => {
    window.ui = SwaggerUIBundle({
      url: '/docs/openapi.yaml',
      dom_id: '#swagger-ui',
      presets: [SwaggerUIBundle.presets.apis],
      layout: "BaseLayout"
    });
  };
</script>
</body>
</html>`
		return c.HTML(http.StatusOK, html)
	})

	// 9. Register API Routes
	deliveryHttp.SetupRoutes(deliveryHttp.RouterConfig{
		Echo:           e,
		TokenManager:   tokenMgr,
		RBACRepo:       rbacRepo,
		AuthHandler:    authHandler,
		ProjectHandler: projectHandler,
		SprintHandler:  sprintHandler,
		IssueHandler:   issueHandler,
		MetricHandler:  metricHandler,
	})

	// 10. Start Server with Graceful Shutdown
	serverAddr := ":" + cfg.App.Port
	go func() {
		logger.Info("HTTP server listening", slog.String("addr", serverAddr), slog.String("env", cfg.App.Env))
		if err := e.Start(serverAddr); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("HTTP server failed to start", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down HTTP server gracefully...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := e.Shutdown(shutdownCtx); err != nil {
		logger.Error("HTTP server graceful shutdown failed", slog.String("error", err.Error()))
	} else {
		logger.Info("HTTP server stopped gracefully")
	}
}
