package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"

	"scout/internal/infrastructure/jwt"
)

// Auth validates the Bearer JWT and stores the user id and role in locals.
// The token may arrive either in the Authorization header or as a `token`
// query parameter (needed for <img src> requests that cannot set headers).
func Auth(jwtSvc *jwt.Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		token := ""
		if authHeader := c.Get(fiber.HeaderAuthorization); authHeader != "" {
			const prefix = "Bearer "
			if !strings.HasPrefix(authHeader, prefix) {
				return fiber.NewError(fiber.StatusUnauthorized, "invalid Authorization header format")
			}
			token = strings.TrimPrefix(authHeader, prefix)
		} else if q := c.Query("token"); q != "" {
			token = q
		}
		if token == "" {
			return fiber.NewError(fiber.StatusUnauthorized, "missing authentication token")
		}
		claims, err := jwtSvc.Validate(token)
		if err != nil {
			return fiber.NewError(fiber.StatusUnauthorized, "invalid or expired token")
		}
		c.Locals(LocalUserID, claims.UserID)
		c.Locals(LocalUserRole, claims.Role)
		return c.Next()
	}
}

// RequireAdmin allows only requests authenticated as an admin.
func RequireAdmin() fiber.Handler {
	return func(c *fiber.Ctx) error {
		role, _ := c.Locals(LocalUserRole).(string)
		if role != "admin" {
			return fiber.NewError(fiber.StatusForbidden, "admin role required")
		}
		return c.Next()
	}
}
