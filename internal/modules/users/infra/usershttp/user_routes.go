package usershttp

import "github.com/gofiber/fiber/v2"

func RegisterRoutes(app *fiber.App, controller *UserController) {
	api := app.Group("/api")
	api.Post("/users", controller.Create)
	api.Get("/users/:id", controller.GetByID)
}
