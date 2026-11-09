package http

import (
	"github.com/labstack/echo/v4"
	"github.com/luqmanulhakimdev/tykket-go/internal/delivery/http/handler"
	"github.com/luqmanulhakimdev/tykket-go/internal/delivery/http/middleware"
	"github.com/luqmanulhakimdev/tykket-go/internal/domain"
	"github.com/luqmanulhakimdev/tykket-go/pkg/token"
)

type RouterConfig struct {
	Echo           *echo.Echo
	TokenManager   token.TokenManager
	RBACRepo       domain.RBACRepository
	AuthHandler    *handler.AuthHandler
	ProjectHandler *handler.ProjectHandler
	SprintHandler  *handler.SprintHandler
	IssueHandler   *handler.IssueHandler
	MetricHandler  *handler.MetricHandler
}

func SetupRoutes(cfg RouterConfig) {
	e := cfg.Echo
	tokenMgr := cfg.TokenManager
	rbacRepo := cfg.RBACRepo

	authMiddleware := middleware.AuthMiddleware(tokenMgr)

	v1 := e.Group("/api/v1")

	// =========================================================================
	// 1. Auth & Identity Routes (Public & Semi-Protected)
	// =========================================================================
	authGroup := v1.Group("/auth")
	authGroup.POST("/register", cfg.AuthHandler.Register)
	authGroup.POST("/login", cfg.AuthHandler.Login)
	authGroup.POST("/refresh", cfg.AuthHandler.RefreshToken)
	authGroup.POST("/google", cfg.AuthHandler.GoogleLogin)
	authGroup.GET("/me", cfg.AuthHandler.GetMe, authMiddleware)

	// Global lookup for issue by key (e.g., /api/v1/issues/TYK-42)
	v1.GET("/issues/:key", cfg.IssueHandler.GetByKey, authMiddleware)

	// =========================================================================
	// 2. Project Routes (Protected)
	// =========================================================================
	projectsGroup := v1.Group("/projects", authMiddleware)
	projectsGroup.GET("", cfg.ProjectHandler.List)
	projectsGroup.POST("", cfg.ProjectHandler.Create)
	projectsGroup.GET("/:projectId", cfg.ProjectHandler.GetByID)
	projectsGroup.PUT("/:projectId", cfg.ProjectHandler.Update, middleware.RequirePermission(rbacRepo, domain.PermProjectUpdate))
	projectsGroup.DELETE("/:projectId", cfg.ProjectHandler.Delete, middleware.RequirePermission(rbacRepo, domain.PermProjectDelete))

	// Project Members
	projectsGroup.GET("/:projectId/members", cfg.ProjectHandler.ListMembers)
	projectsGroup.POST("/:projectId/members", cfg.ProjectHandler.AddMember, middleware.RequirePermission(rbacRepo, domain.PermMemberManage))
	projectsGroup.PUT("/:projectId/members/:userId", cfg.ProjectHandler.UpdateMemberRole, middleware.RequirePermission(rbacRepo, domain.PermMemberManage))
	projectsGroup.DELETE("/:projectId/members/:userId", cfg.ProjectHandler.RemoveMember, middleware.RequirePermission(rbacRepo, domain.PermMemberManage))

	// =========================================================================
	// 3. Sprint & Backlog Routes
	// =========================================================================
	projectsGroup.GET("/:projectId/sprints", cfg.SprintHandler.List)
	projectsGroup.POST("/:projectId/sprints", cfg.SprintHandler.Create, middleware.RequirePermission(rbacRepo, domain.PermSprintManage))
	projectsGroup.GET("/:projectId/sprints/:sprintId", cfg.SprintHandler.GetByID)
	projectsGroup.PUT("/:projectId/sprints/:sprintId", cfg.SprintHandler.Update, middleware.RequirePermission(rbacRepo, domain.PermSprintManage))
	projectsGroup.POST("/:projectId/sprints/:sprintId/start", cfg.SprintHandler.Start, middleware.RequirePermission(rbacRepo, domain.PermSprintManage))
	projectsGroup.POST("/:projectId/sprints/:sprintId/complete", cfg.SprintHandler.Complete, middleware.RequirePermission(rbacRepo, domain.PermSprintManage))
	projectsGroup.DELETE("/:projectId/sprints/:sprintId", cfg.SprintHandler.Delete, middleware.RequirePermission(rbacRepo, domain.PermSprintManage))
	projectsGroup.POST("/:projectId/sprints/bulk-move", cfg.SprintHandler.BulkMove, middleware.RequirePermission(rbacRepo, domain.PermIssueUpdate))

	// =========================================================================
	// 4. Issue Management Routes (Core Ticket Engine)
	// =========================================================================
	projectsGroup.GET("/:projectId/issues", cfg.IssueHandler.List)
	projectsGroup.POST("/:projectId/issues", cfg.IssueHandler.Create, middleware.RequirePermission(rbacRepo, domain.PermIssueCreate))
	projectsGroup.GET("/:projectId/issues/:issueId", cfg.IssueHandler.GetByID)
	projectsGroup.PUT("/:projectId/issues/:issueId", cfg.IssueHandler.Update, middleware.RequirePermission(rbacRepo, domain.PermIssueUpdate))
	projectsGroup.PATCH("/:projectId/issues/:issueId/move", cfg.IssueHandler.Move, middleware.RequirePermission(rbacRepo, domain.PermIssueUpdate))
	projectsGroup.DELETE("/:projectId/issues/:issueId", cfg.IssueHandler.Delete, middleware.RequirePermission(rbacRepo, domain.PermIssueDelete))

	// =========================================================================
	// 5. Metrics, Reporting & Export Routes
	// =========================================================================
	projectsGroup.GET("/:projectId/metrics", cfg.MetricHandler.GetProjectMetrics, middleware.RequirePermission(rbacRepo, domain.PermReportView))
	projectsGroup.GET("/:projectId/sprints/:sprintId/burndown", cfg.MetricHandler.GetSprintBurndown, middleware.RequirePermission(rbacRepo, domain.PermReportView))
	projectsGroup.GET("/:projectId/export/csv", cfg.MetricHandler.ExportCSV, middleware.RequirePermission(rbacRepo, domain.PermReportView))
	projectsGroup.GET("/:projectId/export/summary", cfg.MetricHandler.ExportSummary, middleware.RequirePermission(rbacRepo, domain.PermReportView))
}

