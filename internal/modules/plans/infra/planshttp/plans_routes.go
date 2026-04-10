package planshttp

import (
	"golang_boilerplate_module/internal/modules/auth/infra/authhttp"

	"github.com/gofiber/fiber/v2"
)

// RegisterRoutes registers all plan management HTTP routes on the Fiber app.
// Public routes: list active plans and get plan by slug.
// Admin routes: create, update, and delete plans (requires AuthMiddleware + AdminRequired).
func RegisterRoutes(app *fiber.App, ctrl *PlansController, authMW *authhttp.AuthMiddleware) {
	plans := app.Group("/api/plans")

	// Public routes
	plans.Get("/", ctrl.ListPlans)
	plans.Get("/:slug", ctrl.GetPlan)

	// Admin routes
	admin := plans.Group("", authMW.Required(), AdminRequired())
	admin.Post("/", ctrl.CreatePlan)
	admin.Put("/:id", ctrl.UpdatePlan)
	admin.Delete("/:id", ctrl.DeletePlan)
}
