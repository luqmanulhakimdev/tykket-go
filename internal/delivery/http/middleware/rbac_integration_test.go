package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/luqmanulhakimdev/tykket-go/internal/domain"
)

type mockRBACLiveChecker struct {
	userRoleMap map[uuid.UUID]string // userID -> role name
}

func (m *mockRBACLiveChecker) HasPermission(ctx context.Context, userID, projectID uuid.UUID, permission string) (bool, error) {
	role, ok := m.userRoleMap[userID]
	if !ok {
		return false, nil
	}

	switch role {
	case "OWNER":
		// All permissions
		return true, nil
	case "ADMIN":
		// All except project:delete
		return permission != domain.PermProjectDelete, nil
	case "MEMBER":
		// Can create/update issues and view reports
		return permission == domain.PermIssueCreate || permission == domain.PermIssueUpdate || permission == domain.PermReportView, nil
	case "VIEWER":
		// Read-only metrics and reports
		return permission == domain.PermReportView, nil
	default:
		return false, nil
	}
}

func TestRBAC_RolePermissionMatrix_ViewerStrictlyBlocked(t *testing.T) {
	projectID := uuid.New()

	ownerID := uuid.New()
	adminID := uuid.New()
	memberID := uuid.New()
	viewerID := uuid.New()
	strangerID := uuid.New()

	checker := &mockRBACLiveChecker{
		userRoleMap: map[uuid.UUID]string{
			ownerID:  "OWNER",
			adminID:  "ADMIN",
			memberID: "MEMBER",
			viewerID: "VIEWER",
		},
	}

	e := echo.New()

	testMatrix := []struct {
		name       string
		userID     uuid.UUID
		role       string
		perm       string
		shouldPass bool
	}{
		// 1. VIEWER TESTS (MUST BE STRICTLY BLOCKED FROM ALL MUTATIONS)
		{"Viewer cannot create issue", viewerID, "VIEWER", domain.PermIssueCreate, false},
		{"Viewer cannot update issue", viewerID, "VIEWER", domain.PermIssueUpdate, false},
		{"Viewer cannot delete issue", viewerID, "VIEWER", domain.PermIssueDelete, false},
		{"Viewer cannot manage sprint", viewerID, "VIEWER", domain.PermSprintManage, false},
		{"Viewer cannot manage member", viewerID, "VIEWER", domain.PermMemberManage, false},
		{"Viewer cannot update project", viewerID, "VIEWER", domain.PermProjectUpdate, false},
		{"Viewer cannot delete project", viewerID, "VIEWER", domain.PermProjectDelete, false},
		{"Viewer CAN view report", viewerID, "VIEWER", domain.PermReportView, true},

		// 2. MEMBER TESTS
		{"Member CAN create issue", memberID, "MEMBER", domain.PermIssueCreate, true},
		{"Member CAN update issue", memberID, "MEMBER", domain.PermIssueUpdate, true},
		{"Member CAN view report", memberID, "MEMBER", domain.PermReportView, true},
		{"Member cannot delete issue", memberID, "MEMBER", domain.PermIssueDelete, false},
		{"Member cannot manage sprint", memberID, "MEMBER", domain.PermSprintManage, false},
		{"Member cannot manage member", memberID, "MEMBER", domain.PermMemberManage, false},
		{"Member cannot delete project", memberID, "MEMBER", domain.PermProjectDelete, false},

		// 3. ADMIN TESTS
		{"Admin CAN manage sprint", adminID, "ADMIN", domain.PermSprintManage, true},
		{"Admin CAN manage member", adminID, "ADMIN", domain.PermMemberManage, true},
		{"Admin CAN create issue", adminID, "ADMIN", domain.PermIssueCreate, true},
		{"Admin CAN delete issue", adminID, "ADMIN", domain.PermIssueDelete, true},
		{"Admin cannot delete project", adminID, "ADMIN", domain.PermProjectDelete, false},

		// 4. OWNER TESTS
		{"Owner CAN delete project", ownerID, "OWNER", domain.PermProjectDelete, true},
		{"Owner CAN manage sprint", ownerID, "OWNER", domain.PermSprintManage, true},
		{"Owner CAN create issue", ownerID, "OWNER", domain.PermIssueCreate, true},

		// 5. STRANGER TESTS
		{"Non-member cannot do anything", strangerID, "NONE", domain.PermReportView, false},
	}

	for _, tc := range testMatrix {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+projectID.String()+"/action", nil)
			c := e.NewContext(req, rec)
			c.Set(CtxKeyUserID, tc.userID)
			c.SetParamNames("projectId")
			c.SetParamValues(projectID.String())

			handler := RequirePermission(checker, tc.perm)(func(c echo.Context) error {
				return c.String(http.StatusOK, "success")
			})

			_ = handler(c)

			if tc.shouldPass {
				if rec.Code != http.StatusOK {
					t.Errorf("expected 200 OK for role %s on perm %s, got %d (body: %s)", tc.role, tc.perm, rec.Code, rec.Body.String())
				}
			} else {
				if rec.Code != http.StatusForbidden {
					t.Errorf("expected 403 Forbidden for role %s on perm %s, got %d (body: %s)", tc.role, tc.perm, rec.Code, rec.Body.String())
				}
			}
		})
	}
}
