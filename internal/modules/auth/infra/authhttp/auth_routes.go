package authhttp

import "github.com/gofiber/fiber/v2"

// RegisterRoutes registers all authentication HTTP routes on the Fiber app.
// Public routes: register, login, refresh, confirm-email, forgot-password, reset-password.
// Protected routes: logout, change-password (require AuthMiddleware).
func RegisterRoutes(app *fiber.App, ctrl *AuthController, mw *AuthMiddleware) {
	auth := app.Group("/auth")
	auth.Post("/register", ctrl.Register)
	auth.Post("/login", ctrl.Login)
	auth.Post("/refresh", ctrl.RefreshToken)
	auth.Post("/confirm-email", ctrl.ConfirmEmail)
	auth.Get("/confirm-email", ctrl.ConfirmEmail)
	auth.Post("/forgot-password", ctrl.ForgotPassword)
	auth.Post("/reset-password", ctrl.ResetPassword)
	auth.Post("/logout", mw.Required(), ctrl.Logout)
	auth.Put("/change-password", mw.Required(), ctrl.ChangePassword)
}
