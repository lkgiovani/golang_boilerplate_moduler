package planshttp

import (
	"golang_boilerplate_module/internal/shared/domain/exceptions"

	"github.com/gofiber/fiber/v2"
)

// AdminRequired returns a Fiber middleware handler that enforces admin-only access.
// It reads the "userAdmin" local (set by AuthMiddleware) and returns 403 if the
// user is not an admin.
func AdminRequired() fiber.Handler {
	return func(c *fiber.Ctx) error {
		isAdmin, ok := c.Locals("userAdmin").(bool)
		if !ok || !isAdmin {
			return exceptions.NewForbiddenException("admin access required", nil)
		}
		return c.Next()
	}
}
