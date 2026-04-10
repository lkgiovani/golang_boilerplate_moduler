package planshttp

import (
	"golang_boilerplate_module/internal/modules/auth/infra/authhttp"

	"github.com/gofiber/fiber/v2"
)

// RegisterRoutes registers all plan, subscription, and webhook HTTP routes on the Fiber app.
// Public routes: list active plans, get plan by slug.
// Admin routes: create, update, and delete plans (requires AuthMiddleware + AdminRequired).
// Subscription routes: checkout, get current subscription, cancel (requires AuthMiddleware).
// Webhook route: Stripe webhook (no auth, uses Stripe signature verification).
func RegisterRoutes(app *fiber.App, planCtrl *PlansController, webhookCtrl *WebhookController, authMW *authhttp.AuthMiddleware) {
	// Plan routes (public)
	plans := app.Group("/api/plans")
	plans.Get("/", planCtrl.ListPlans)
	plans.Get("/:slug", planCtrl.GetPlan)

	// Plan routes (admin only)
	admin := plans.Group("", authMW.Required(), AdminRequired())
	admin.Post("/", planCtrl.CreatePlan)
	admin.Put("/:id", planCtrl.UpdatePlan)
	admin.Delete("/:id", planCtrl.DeletePlan)

	// Subscription routes (authenticated)
	subs := app.Group("/api/subscriptions", authMW.Required())
	subs.Post("/checkout", planCtrl.Subscribe)
	subs.Get("/me", planCtrl.GetSubscription)
	subs.Post("/cancel", planCtrl.CancelSubscription)

	// Webhook route (no auth - Stripe signature verification)
	app.Post("/api/webhooks/stripe", webhookCtrl.HandleWebhook)
}
