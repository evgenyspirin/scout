package rest_test

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"scout/internal/interface/api/rest"
)

// TestPanicRecovery verifies the global panic-recovery mechanism: a handler
// that panics must not crash the process and must return a shaped 500 error.
func TestPanicRecovery(t *testing.T) {
	app := fiber.New(fiber.Config{ErrorHandler: rest.ErrorHandler(zap.NewNop())})
	app.Use(recover.New())
	app.Get("/boom", func(_ *fiber.Ctx) error {
		panic("kaboom")
	})

	req := httptest.NewRequest("GET", "/boom", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
}
