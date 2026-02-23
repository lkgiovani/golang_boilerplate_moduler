package http

import "github.com/gofiber/fiber/v2"

func RegisterRoutes(app *fiber.App, controller *HealthController) {
	app.Get("/healthz", controller.CheckHealth)
	app.Get("/readyz", controller.CheckReadiness)
}