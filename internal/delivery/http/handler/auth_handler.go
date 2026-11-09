package handler

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/luqmanulhakimdev/tykket-go/internal/delivery/http/middleware"
	"github.com/luqmanulhakimdev/tykket-go/internal/domain"
	"github.com/luqmanulhakimdev/tykket-go/pkg/response"
)

type AuthHandler struct {
	authUsecase domain.AuthUsecase
}

func NewAuthHandler(authUsecase domain.AuthUsecase) *AuthHandler {
	return &AuthHandler{authUsecase: authUsecase}
}

func (h *AuthHandler) Register(c echo.Context) error {
	var req domain.RegisterRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "Invalid request payload", err)
	}
	if err := c.Validate(&req); err != nil {
		return response.BadRequest(c, err.Error(), nil)
	}

	result, err := h.authUsecase.Register(c.Request().Context(), req)
	if err != nil {
		if errors.Is(err, domain.ErrConflict) {
			return response.Conflict(c, "Email is already registered")
		}
		return response.InternalServerError(c, err.Error())
	}

	return response.Created(c, "User registered successfully", result)
}

func (h *AuthHandler) Login(c echo.Context) error {
	var req domain.LoginRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "Invalid request payload", err)
	}
	if err := c.Validate(&req); err != nil {
		return response.BadRequest(c, err.Error(), nil)
	}

	result, err := h.authUsecase.Login(c.Request().Context(), req)
	if err != nil {
		if errors.Is(err, domain.ErrUnauthorized) {
			return response.Unauthorized(c, "Invalid email or password")
		}
		return response.InternalServerError(c, err.Error())
	}

	return response.OK(c, "Login successful", result)
}

func (h *AuthHandler) RefreshToken(c echo.Context) error {
	var req domain.RefreshTokenRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "Invalid request payload", err)
	}
	if err := c.Validate(&req); err != nil {
		return response.BadRequest(c, err.Error(), nil)
	}

	result, err := h.authUsecase.RefreshToken(c.Request().Context(), req)
	if err != nil {
		return response.Unauthorized(c, "Invalid or expired refresh token")
	}

	return response.OK(c, "Token refreshed successfully", result)
}

func (h *AuthHandler) GoogleLogin(c echo.Context) error {
	var req domain.GoogleAuthRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "Invalid request payload", err)
	}
	if err := c.Validate(&req); err != nil {
		return response.BadRequest(c, err.Error(), nil)
	}

	result, err := h.authUsecase.GoogleLogin(c.Request().Context(), req)
	if err != nil {
		return response.Error(c, http.StatusUnauthorized, "Google authentication failed", err)
	}

	return response.OK(c, "Google login successful", result)
}

func (h *AuthHandler) GetMe(c echo.Context) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return response.Unauthorized(c, "Unauthorized")
	}

	user, err := h.authUsecase.GetMe(c.Request().Context(), userID)
	if err != nil {
		return response.NotFound(c, "User not found")
	}

	return response.OK(c, "Profile retrieved", user)
}

