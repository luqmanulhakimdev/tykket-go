package middleware

import (
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/luqmanulhakimdev/tykket-go/pkg/response"
	"github.com/luqmanulhakimdev/tykket-go/pkg/token"
)

const (
	CtxKeyUserID = "user_id"
	CtxKeyEmail  = "email"
	CtxKeyClaims = "claims"
)

var (
	ErrUnauthorized = errors.New("unauthorized")
)

// AuthMiddleware creates a middleware that validates Bearer JWT access tokens.
func AuthMiddleware(tokenMgr token.TokenManager) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return response.Unauthorized(c, "Missing Authorization header")
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				return response.Unauthorized(c, "Invalid Authorization header format. Format: Bearer <token>")
			}

			claims, err := tokenMgr.ValidateToken(parts[1], token.AccessToken)
			if err != nil {
				return response.Unauthorized(c, "Invalid or expired token")
			}

			c.Set(CtxKeyUserID, claims.UserID)
			c.Set(CtxKeyEmail, claims.Email)
			c.Set(CtxKeyClaims, claims)

			return next(c)
		}
	}
}

// GetUserID extracts the user UUID from echo context.
func GetUserID(c echo.Context) (uuid.UUID, error) {
	val := c.Get(CtxKeyUserID)
	if val == nil {
		return uuid.Nil, ErrUnauthorized
	}
	id, ok := val.(uuid.UUID)
	if !ok {
		return uuid.Nil, ErrUnauthorized
	}
	return id, nil
}

// GetUserEmail extracts the user email from echo context.
func GetUserEmail(c echo.Context) (string, error) {
	val := c.Get(CtxKeyEmail)
	if val == nil {
		return "", ErrUnauthorized
	}
	email, ok := val.(string)
	if !ok {
		return "", ErrUnauthorized
	}
	return email, nil
}

