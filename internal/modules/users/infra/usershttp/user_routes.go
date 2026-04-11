package usershttp

import (
	"golang_boilerplate_module/internal/modules/auth/infra/authhttp"

	"github.com/gofiber/fiber/v2"
)

func RegisterRoutes(app *fiber.App, controller *UserController, mw *authhttp.AuthMiddleware) {
	api := app.Group("/api")
	api.Post("/users", controller.Create)
	api.Get("/users/me", mw.Required(), controller.GetMe)
	api.Put("/users/me", mw.Required(), controller.UpdateMe)
	api.Get("/users/:id", controller.GetByID)
}
