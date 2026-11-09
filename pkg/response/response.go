package response

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// Response defines the standardized JSON envelope.
type Response struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Meta    interface{} `json:"meta,omitempty"`
	Error   interface{} `json:"error,omitempty"`
}

// Meta defines pagination or supplemental metadata.
type Meta struct {
	Page       int   `json:"page,omitempty"`
	Limit      int   `json:"limit,omitempty"`
	Total      int64 `json:"total,omitempty"`
	TotalPages int   `json:"total_pages,omitempty"`
}

// JSON sends a standardized JSON response.
func JSON(c echo.Context, statusCode int, success bool, message string, data, meta, err interface{}) error {
	return c.JSON(statusCode, Response{
		Success: success,
		Message: message,
		Data:    data,
		Meta:    meta,
		Error:   err,
	})
}

// OK sends a 200 OK response with data.
func OK(c echo.Context, message string, data interface{}) error {
	return JSON(c, http.StatusOK, true, message, data, nil, nil)
}

// OKWithMeta sends a 200 OK response with data and pagination meta.
func OKWithMeta(c echo.Context, message string, data interface{}, meta *Meta) error {
	return JSON(c, http.StatusOK, true, message, data, meta, nil)
}

// Created sends a 201 Created response.
func Created(c echo.Context, message string, data interface{}) error {
	return JSON(c, http.StatusCreated, true, message, data, nil, nil)
}

// NoContent sends a 204 No Content response.
func NoContent(c echo.Context) error {
	return c.NoContent(http.StatusNoContent)
}

// Error sends an error response.
func Error(c echo.Context, statusCode int, message string, err interface{}) error {
	var errPayload interface{} = err
	if e, ok := err.(error); ok && e != nil {
		errPayload = e.Error()
	}
	return JSON(c, statusCode, false, message, nil, nil, errPayload)
}

// BadRequest sends a 400 Bad Request error.
func BadRequest(c echo.Context, message string, err interface{}) error {
	if message == "" {
		message = "Invalid request payload"
	}
	return Error(c, http.StatusBadRequest, message, err)
}

// Unauthorized sends a 401 Unauthorized error.
func Unauthorized(c echo.Context, message string) error {
	if message == "" {
		message = "Unauthorized access"
	}
	return Error(c, http.StatusUnauthorized, message, nil)
}

// Forbidden sends a 403 Forbidden error.
func Forbidden(c echo.Context, message string) error {
	if message == "" {
		message = "Forbidden access"
	}
	return Error(c, http.StatusForbidden, message, nil)
}

// NotFound sends a 404 Not Found error.
func NotFound(c echo.Context, message string) error {
	if message == "" {
		message = "Resource not found"
	}
	return Error(c, http.StatusNotFound, message, nil)
}

// Conflict sends a 409 Conflict error.
func Conflict(c echo.Context, message string) error {
	if message == "" {
		message = "Resource conflict"
	}
	return Error(c, http.StatusConflict, message, nil)
}

// InternalServerError sends a 500 Internal Server Error.
func InternalServerError(c echo.Context, message string) error {
	if message == "" {
		message = "Internal server error"
	}
	return Error(c, http.StatusInternalServerError, message, nil)
}

