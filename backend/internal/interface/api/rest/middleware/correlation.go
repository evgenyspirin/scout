package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// Context locals keys.
const (
	LocalRequestID = "request_id"
	LocalUserID    = "user_id"
	LocalUserRole  = "user_role"

	headerRequestID = "X-Request-ID"
)

// Correlation ensures every request has a correlation id, echoed in the response.
func Correlation() fiber.Handler {
	return func(c *fiber.Ctx) error {
		rid := c.Get(headerRequestID)
		if rid == "" {
			rid = uuid.NewString()
		}
		c.Locals(LocalRequestID, rid)
		c.Set(headerRequestID, rid)
		return c.Next()
	}
}
