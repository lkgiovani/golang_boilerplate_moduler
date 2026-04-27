package planshttp

import (
	"golang_boilerplate_module/internal/modules/auth/infra/authhttp"

	"github.com/gofiber/fiber/v2"
)

// RegisterRoutes registers all plan, subscription, and webhook HTTP routes on the Fiber app.
// Webhook route uses :gateway path param to support multiple gateways without code changes.
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
	subs.Post("/me/payment-method", planCtrl.UpdatePaymentMethod)
	subs.Get("/me/status", planCtrl.GetSubscriptionStatus)
	subs.Post("/me/portal", planCtrl.CreateBillingPortal)

	// Refund — admin-only per D-20
	subsAdmin := subs.Group("", AdminRequired())
	subsAdmin.Post("/:id/refund", planCtrl.RefundPayment)

	// Webhook route (parametric, no auth — gateway signature verification)
	app.Post("/api/webhooks/:gateway", webhookCtrl.HandleWebhook)
}
