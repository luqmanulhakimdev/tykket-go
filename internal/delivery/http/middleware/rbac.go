package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/luqmanulhakimdev/tykket-go/pkg/response"
)

// PermissionChecker interface abstracts checking whether a user has a specific permission in a project.
type PermissionChecker interface {
	HasPermission(ctx context.Context, userID, projectID uuid.UUID, permissionCode string) (bool, error)
}

// RequirePermission checks whether the authenticated user has the specified permission for the project.
// The project ID is resolved from path parameter paramName (defaulting to "projectId", falling back to "id").
func RequirePermission(checker PermissionChecker, permissionCode string, paramNames ...string) echo.MiddlewareFunc {
	paramKey := "projectId"
	if len(paramNames) > 0 && paramNames[0] != "" {
		paramKey = paramNames[0]
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			userID, err := GetUserID(c)
			if err != nil {
				return response.Unauthorized(c, "Authentication required")
			}

			projectIDStr := c.Param(paramKey)
			if projectIDStr == "" {
				projectIDStr = c.Param("id")
			}
			if projectIDStr == "" {
				return response.BadRequest(c, "Missing project identifier in route", nil)
			}

			projectID, err := uuid.Parse(projectIDStr)
			if err != nil {
				return response.BadRequest(c, "Invalid project ID format", nil)
			}

			// Store project ID in context for downstream handlers
			c.Set("project_id", projectID)

			hasPerm, err := checker.HasPermission(c.Request().Context(), userID, projectID, permissionCode)
			if err != nil {
				return response.Error(c, http.StatusInternalServerError, "Failed to verify permissions", err)
			}

			if !hasPerm {
				return response.Forbidden(c, "You do not have permission to perform this action in this project")
			}

			return next(c)
		}
	}
}

