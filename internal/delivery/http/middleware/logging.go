package middleware

import (
	"log/slog"
	"time"

	"github.com/labstack/echo/v4"
)

// StructuredLogger logs HTTP requests using standard slog.
func StructuredLogger(logger *slog.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			req := c.Request()
			res := c.Response()

			err := next(c)
			if err != nil {
				c.Error(err)
			}

			duration := time.Since(start)
			status := res.Status
			requestID := res.Header().Get(echo.HeaderXRequestID)
			if requestID == "" {
				requestID = req.Header.Get(echo.HeaderXRequestID)
			}

			attrs := []any{
				slog.String("request_id", requestID),
				slog.String("method", req.Method),
				slog.String("uri", req.RequestURI),
				slog.Int("status", status),
				slog.Duration("latency", duration),
				slog.String("remote_ip", c.RealIP()),
				slog.Int64("bytes_out", res.Size),
			}

			if err != nil {
				attrs = append(attrs, slog.String("error", err.Error()))
				logger.Error("HTTP request failed", attrs...)
			} else if status >= 500 {
				logger.Error("HTTP server error", attrs...)
			} else if status >= 400 {
				logger.Warn("HTTP client error", attrs...)
			} else {
				logger.Info("HTTP request processed", attrs...)
			}

			return nil
		}
	}
}

