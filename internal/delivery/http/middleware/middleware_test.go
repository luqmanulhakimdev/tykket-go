package middleware

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/luqmanulhakimdev/tykket-go/pkg/token"
)

type mockTokenMgr struct {
	token.TokenManager
	validClaims *token.Claims
	err         error
}

func (m *mockTokenMgr) ValidateToken(tokenString string, expectedType token.TokenType) (*token.Claims, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.validClaims, nil
}

type mockPermissionChecker struct {
	hasPerm bool
	err     error
}

func (m *mockPermissionChecker) HasPermission(ctx context.Context, userID, projectID uuid.UUID, perm string) (bool, error) {
	return m.hasPerm, m.err
}

func TestAuthMiddleware(t *testing.T) {
	e := echo.New()
	userID := uuid.New()
	email := "test@example.com"

	t.Run("Missing Authorization header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		mw := AuthMiddleware(&mockTokenMgr{})
		handler := mw(func(c echo.Context) error {
			return c.String(http.StatusOK, "ok")
		})

		_ = handler(c)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", rec.Code)
		}
	})

	t.Run("Invalid header format (not Bearer)", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Basic 123456")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		mw := AuthMiddleware(&mockTokenMgr{})
		handler := mw(func(c echo.Context) error {
			return c.String(http.StatusOK, "ok")
		})

		_ = handler(c)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", rec.Code)
		}
	})

	t.Run("Invalid or expired token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer invalidtoken")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		mw := AuthMiddleware(&mockTokenMgr{err: errors.New("expired")})
		handler := mw(func(c echo.Context) error {
			return c.String(http.StatusOK, "ok")
		})

		_ = handler(c)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", rec.Code)
		}
	})

	t.Run("Valid token sets context and proceeds", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer validtoken")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		claims := &token.Claims{
			UserID: userID,
			Email:  email,
			Type:   token.AccessToken,
		}

		mw := AuthMiddleware(&mockTokenMgr{validClaims: claims})
		handler := mw(func(c echo.Context) error {
			extractedID, err := GetUserID(c)
			if err != nil || extractedID != userID {
				t.Errorf("failed to extract user ID: %v", err)
			}
			extractedEmail, err := GetUserEmail(c)
			if err != nil || extractedEmail != email {
				t.Errorf("failed to extract user email: %v", err)
			}
			return c.String(http.StatusOK, "ok")
		})

		_ = handler(c)
		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("GetUserID and GetUserEmail error when missing in context", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		_, err := GetUserID(c)
		if err != ErrUnauthorized {
			t.Errorf("expected ErrUnauthorized, got %v", err)
		}

		_, err = GetUserEmail(c)
		if err != ErrUnauthorized {
			t.Errorf("expected ErrUnauthorized, got %v", err)
		}

		// Wrong type in context
		c.Set(CtxKeyUserID, "not-a-uuid")
		_, err = GetUserID(c)
		if err != ErrUnauthorized {
			t.Errorf("expected ErrUnauthorized for bad type, got %v", err)
		}

		c.Set(CtxKeyEmail, 12345)
		_, err = GetUserEmail(c)
		if err != ErrUnauthorized {
			t.Errorf("expected ErrUnauthorized for bad email type, got %v", err)
		}
	})
}

func TestRequirePermissionMiddleware(t *testing.T) {
	e := echo.New()
	userID := uuid.New()
	projectID := uuid.New()

	t.Run("Unauthorized user", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		mw := RequirePermission(&mockPermissionChecker{}, "issue:create")
		handler := mw(func(c echo.Context) error { return nil })
		_ = handler(c)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", rec.Code)
		}
	})

	t.Run("Missing project id in path", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set(CtxKeyUserID, userID)

		mw := RequirePermission(&mockPermissionChecker{}, "issue:create")
		handler := mw(func(c echo.Context) error { return nil })
		_ = handler(c)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("Invalid project UUID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/projects/bad-uuid", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set(CtxKeyUserID, userID)
		c.SetParamNames("projectId")
		c.SetParamValues("not-a-valid-uuid")

		mw := RequirePermission(&mockPermissionChecker{}, "issue:create")
		handler := mw(func(c echo.Context) error { return nil })
		_ = handler(c)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("Permission check internal error", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set(CtxKeyUserID, userID)
		c.SetParamNames("projectId")
		c.SetParamValues(projectID.String())

		checker := &mockPermissionChecker{err: errors.New("db error")}
		mw := RequirePermission(checker, "issue:create")
		handler := mw(func(c echo.Context) error { return nil })
		_ = handler(c)

		if rec.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", rec.Code)
		}
	})

	t.Run("Forbidden when user lacks permission", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set(CtxKeyUserID, userID)
		c.SetParamNames("projectId")
		c.SetParamValues(projectID.String())

		checker := &mockPermissionChecker{hasPerm: false}
		mw := RequirePermission(checker, "issue:create")
		handler := mw(func(c echo.Context) error { return nil })
		_ = handler(c)

		if rec.Code != http.StatusForbidden {
			t.Errorf("expected 403, got %d", rec.Code)
		}
	})

	t.Run("Allowed when user has permission", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set(CtxKeyUserID, userID)
		c.SetParamNames("projectId")
		c.SetParamValues(projectID.String())

		checker := &mockPermissionChecker{hasPerm: true}
		mw := RequirePermission(checker, "issue:create")
		handler := mw(func(c echo.Context) error {
			return c.String(http.StatusOK, "ok")
		})
		_ = handler(c)

		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}
	})
}

func TestRecoverAndLoggerMiddleware(t *testing.T) {
	e := echo.New()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("Recover middleware handles panic", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/panic", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		mw := Recover(logger)
		handler := mw(func(c echo.Context) error {
			panic("something went catastrophically wrong")
		})

		_ = handler(c)
		if rec.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", rec.Code)
		}
	})

	t.Run("StructuredLogger logs requests with various statuses", func(t *testing.T) {
		statuses := []int{http.StatusOK, http.StatusBadRequest, http.StatusInternalServerError}
		for _, status := range statuses {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			mw := StructuredLogger(logger)
			handler := mw(func(c echo.Context) error {
				time.Sleep(1 * time.Millisecond)
				return c.NoContent(status)
			})

			_ = handler(c)
		}
	})
}
