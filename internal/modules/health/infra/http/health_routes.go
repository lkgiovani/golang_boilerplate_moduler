package http

import "github.com/gofiber/fiber/v2"

// RegisterRoutes mounts health check endpoints on the Fiber app.
// Called via fx.Invoke — side-effect registration at startup.
func RegisterRoutes(app *fiber.App, controller *HealthController) {
	app.Get("/healthz", controller.CheckHealth)
	app.Get("/readyz", controller.CheckReadiness)
}
