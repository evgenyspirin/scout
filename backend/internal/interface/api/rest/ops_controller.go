package rest

import (
	"expvar"

	"github.com/gofiber/adaptor/v2"
	"github.com/gofiber/fiber/v2"

	"scout/internal/infrastructure/metrics"
)

// OpsController exposes health, metrics and debug endpoints.
type OpsController struct {
	metrics *metrics.Metrics
}

// NewOpsController builds an OpsController.
func NewOpsController(m *metrics.Metrics) *OpsController {
	return &OpsController{metrics: m}
}

// Health is a public liveness probe.
func (o *OpsController) Health(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"status": "ok"})
}

// Metrics serves Prometheus metrics (mounted behind admin auth).
func (o *OpsController) Metrics() fiber.Handler {
	return adaptor.HTTPHandler(o.metrics.HTTPHandler())
}

// DebugVars serves Go expvar data (mounted behind admin auth).
func (o *OpsController) DebugVars() fiber.Handler {
	return adaptor.HTTPHandler(expvar.Handler())
}
