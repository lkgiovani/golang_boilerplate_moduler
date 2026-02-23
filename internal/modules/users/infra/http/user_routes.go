package http

import "github.com/gofiber/fiber/v2"

// RegisterRoutes mounts user endpoints on the /api group.
// Called via fx.Invoke — side-effect registration at startup.
func RegisterRoutes(app *fiber.App, controller *UserController) {
	api := app.Group("/api")
	api.Post("/users", controller.Create)
	api.Get("/users/:id", controller.GetByID)
}
