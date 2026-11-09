package response

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestResponseJSON(t *testing.T) {
	e := echo.New()

	t.Run("OK response envelope", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		data := map[string]string{"foo": "bar"}
		if err := OK(c, "success message", data); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}

		var resp Response
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if !resp.Success {
			t.Errorf("expected success true, got false")
		}
	})

	t.Run("OKWithMeta response envelope", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		meta := &Meta{Page: 1, Limit: 10, Total: 100, TotalPages: 10}
		if err := OKWithMeta(c, "list message", []string{"a", "b"}, meta); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("Created response envelope", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		if err := Created(c, "resource created", map[string]int{"id": 1}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if rec.Code != http.StatusCreated {
			t.Errorf("expected 201, got %d", rec.Code)
		}
	})

	t.Run("NoContent response", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		if err := NoContent(c); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if rec.Code != http.StatusNoContent {
			t.Errorf("expected 204, got %d", rec.Code)
		}
	})

	t.Run("BadRequest with default message and error instance", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		rawErr := errors.New("underlying failure")
		if err := BadRequest(c, "", rawErr); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}

		var resp Response
		_ = json.Unmarshal(rec.Body.Bytes(), &resp)
		if resp.Message != "Invalid request payload" {
			t.Errorf("expected default message, got %s", resp.Message)
		}
		if resp.Error != "underlying failure" {
			t.Errorf("expected underlying failure string, got %v", resp.Error)
		}
	})

	t.Run("Unauthorized response with default message", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		if err := Unauthorized(c, ""); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", rec.Code)
		}
	})

	t.Run("Unauthorized response with custom message", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		if err := Unauthorized(c, "Token missing"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", rec.Code)
		}
	})

	t.Run("Forbidden response with default and custom message", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		if err := Forbidden(c, ""); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if rec.Code != http.StatusForbidden {
			t.Errorf("expected 403, got %d", rec.Code)
		}

		rec2 := httptest.NewRecorder()
		c2 := e.NewContext(req, rec2)
		if err := Forbidden(c2, "Custom forbidden"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if rec2.Code != http.StatusForbidden {
			t.Errorf("expected 403, got %d", rec2.Code)
		}
	})

	t.Run("NotFound response with default and custom message", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		if err := NotFound(c, ""); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if rec.Code != http.StatusNotFound {
			t.Errorf("expected 404, got %d", rec.Code)
		}

		rec2 := httptest.NewRecorder()
		c2 := e.NewContext(req, rec2)
		if err := NotFound(c2, "Custom not found"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if rec2.Code != http.StatusNotFound {
			t.Errorf("expected 404, got %d", rec2.Code)
		}
	})

	t.Run("Conflict response with default and custom message", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		if err := Conflict(c, ""); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if rec.Code != http.StatusConflict {
			t.Errorf("expected 409, got %d", rec.Code)
		}

		rec2 := httptest.NewRecorder()
		c2 := e.NewContext(req, rec2)
		if err := Conflict(c2, "Custom conflict"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if rec2.Code != http.StatusConflict {
			t.Errorf("expected 409, got %d", rec2.Code)
		}
	})

	t.Run("InternalServerError with default and custom message", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		if err := InternalServerError(c, ""); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if rec.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", rec.Code)
		}

		rec2 := httptest.NewRecorder()
		c2 := e.NewContext(req, rec2)
		if err := InternalServerError(c2, "Custom 500"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if rec2.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", rec2.Code)
		}
	})
}
