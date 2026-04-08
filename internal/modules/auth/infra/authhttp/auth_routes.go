package authhttp

import "github.com/gofiber/fiber/v2"

// RegisterRoutes registers all authentication HTTP routes on the Fiber app.
// Public routes: register, login, refresh.
// Protected routes: logout (requires AuthMiddleware).
func RegisterRoutes(app *fiber.App, ctrl *AuthController, mw *AuthMiddleware) {
	auth := app.Group("/auth")
	auth.Post("/register", ctrl.Register)
	auth.Post("/login", ctrl.Login)
	auth.Post("/refresh", ctrl.RefreshToken)
	auth.Post("/logout", mw.Required(), ctrl.Logout)
}
