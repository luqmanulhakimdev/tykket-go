package middleware

import (
	"fmt"
	"log/slog"
	"runtime/debug"

	"github.com/labstack/echo/v4"
	"github.com/luqmanulhakimdev/tykket-go/pkg/response"
)

// Recover catches any panic, logs the stack trace, and sends a standard 500 JSON response.
func Recover(logger *slog.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			defer func() {
				if r := recover(); r != nil {
					err, ok := r.(error)
					if !ok {
						err = fmt.Errorf("%v", r)
					}

					stack := string(debug.Stack())
					logger.Error("panic recovered",
						slog.String("error", err.Error()),
						slog.String("stack", stack),
						slog.String("uri", c.Request().RequestURI),
					)

					if !c.Response().Committed {
						_ = response.InternalServerError(c, "An unexpected error occurred")
					}
				}
			}()
			return next(c)
		}
	}
}

