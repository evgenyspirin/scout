package middleware

import (
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"scout/internal/infrastructure/metrics"
)

// RequestLogger logs every request with a correlation id and records metrics.
// It never logs secrets, tokens or request bodies.
func RequestLogger(logger *zap.Logger, m *metrics.Metrics) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		err := c.Next()
		dur := time.Since(start)

		status := c.Response().StatusCode()
		// Fiber's c.Method()/c.Path() return strings backed by a pooled fasthttp
		// buffer that is recycled after the response is written. We store these
		// as long-lived Prometheus label keys, so they MUST be cloned first to
		// avoid the buffer being mutated under us (which corrupts label maps).
		method := strings.Clone(c.Method())
		// Use the route template (not the concrete URL) to keep metric/label cardinality low.
		routePath := c.Route().Path
		if routePath == "" {
			routePath = strings.Clone(c.Path())
		}

		if m != nil {
			m.ObserveHTTP(method, routePath, status, dur)
		}

		rid, _ := c.Locals(LocalRequestID).(string)
		fields := []zap.Field{
			zap.String("request_id", rid),
			zap.String("method", method),
			zap.String("path", c.Path()),
			zap.Int("status", status),
			zap.Duration("duration", dur),
			zap.String("client_ip", c.IP()),
			zap.String("user_agent", c.Get(fiber.HeaderUserAgent)),
		}

		switch {
		case status >= fiber.StatusInternalServerError:
			logger.Error("http request", fields...)
		case status >= fiber.StatusBadRequest:
			logger.Warn("http request", fields...)
		default:
			logger.Info("http request", fields...)
		}
		return err
	}
}
