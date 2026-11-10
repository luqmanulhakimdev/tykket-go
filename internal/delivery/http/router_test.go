package http

import (
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/luqmanulhakimdev/tykket-go/internal/delivery/http/handler"
	"github.com/luqmanulhakimdev/tykket-go/pkg/token"
)

func TestSetupRoutes(t *testing.T) {
	e := echo.New()
	tokenMgr := token.NewJWTManager("secret", 0, 0)

	cfg := RouterConfig{
		Echo:           e,
		TokenManager:   tokenMgr,
		RBACRepo:       nil,
		AuthHandler:    &handler.AuthHandler{},
		ProjectHandler: &handler.ProjectHandler{},
		SprintHandler:  &handler.SprintHandler{},
		IssueHandler:   &handler.IssueHandler{},
		MetricHandler:  &handler.MetricHandler{},
		UploadHandler:  handler.NewUploadHandler("uploads"),
		CommentHandler: &handler.CommentHandler{},
	}

	SetupRoutes(cfg)

	routes := e.Routes()
	if len(routes) == 0 {
		t.Fatalf("expected routes registered, got 0")
	}

	foundRegister := false
	foundProjects := false
	foundIssues := false
	foundUpload := false
	foundComments := false
	for _, r := range routes {
		if r.Path == "/api/v1/auth/register" {
			foundRegister = true
		}
		if r.Path == "/api/v1/projects" {
			foundProjects = true
		}
		if r.Path == "/api/v1/projects/:projectId/issues" {
			foundIssues = true
		}
		if r.Path == "/api/v1/upload" {
			foundUpload = true
		}
		if r.Path == "/api/v1/issues/:issueId/comments" {
			foundComments = true
		}
	}

	if !foundRegister || !foundProjects || !foundIssues || !foundUpload || !foundComments {
		t.Errorf("missing expected routes in Echo instance: register=%v, projects=%v, issues=%v, upload=%v, comments=%v",
			foundRegister, foundProjects, foundIssues, foundUpload, foundComments)
	}
}
