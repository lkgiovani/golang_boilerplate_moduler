package authhttp

import (
	"time"

	"golang_boilerplate_module/internal/config"
	"golang_boilerplate_module/internal/modules/auth/application/authusecases"
	"golang_boilerplate_module/internal/modules/auth/authdomain"
	"golang_boilerplate_module/internal/shared/domain/providers"
	"golang_boilerplate_module/internal/shared/infra/http/middleware"
	"golang_boilerplate_module/internal/shared/infra/observability"

	"github.com/gofiber/fiber/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

var tracer = otel.Tracer("auth.http")

// AuthController handles HTTP requests for authentication endpoints.
type AuthController struct {
	registerUC       *authusecases.RegisterUseCase
	loginUC          *authusecases.LoginUseCase
	refreshTokenUC   *authusecases.RefreshTokenUseCase
	logoutUC         *authusecases.LogoutUseCase
	confirmEmailUC   *authusecases.ConfirmEmailUseCase
	forgotPasswordUC *authusecases.ForgotPasswordUseCase
	resetPasswordUC  *authusecases.ResetPasswordUseCase
	changePasswordUC *authusecases.ChangePasswordUseCase
	cfg              *config.Config
	logger           providers.LoggerProvider
}

// NewAuthController creates a new AuthController with all required dependencies.
func NewAuthController(
	registerUC *authusecases.RegisterUseCase,
	loginUC *authusecases.LoginUseCase,
	refreshTokenUC *authusecases.RefreshTokenUseCase,
	logoutUC *authusecases.LogoutUseCase,
	confirmEmailUC *authusecases.ConfirmEmailUseCase,
	forgotPasswordUC *authusecases.ForgotPasswordUseCase,
	resetPasswordUC *authusecases.ResetPasswordUseCase,
	changePasswordUC *authusecases.ChangePasswordUseCase,
	cfg *config.Config,
	logger providers.LoggerProvider,
) *AuthController {
	return &AuthController{
		registerUC:       registerUC,
		loginUC:          loginUC,
		refreshTokenUC:   refreshTokenUC,
		logoutUC:         logoutUC,
		confirmEmailUC:   confirmEmailUC,
		forgotPasswordUC: forgotPasswordUC,
		resetPasswordUC:  resetPasswordUC,
		changePasswordUC: changePasswordUC,
		cfg:              cfg,
		logger:           logger,
	}
}

// registerRequest is the expected JSON body for registration.
type registerRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
	DeviceID string `json:"device_id"`
}

// loginRequest is the expected JSON body for login.
type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	DeviceID string `json:"device_id"`
}

// refreshRequest is the expected JSON body for token refresh.
type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
	DeviceID     string `json:"device_id"`
}

// confirmEmailRequest is the body for POST /auth/confirm-email.
type confirmEmailRequest struct {
	Token string `json:"token"`
}

// forgotPasswordRequest is the body for POST /auth/forgot-password.
type forgotPasswordRequest struct {
	Email string `json:"email"`
}

// resetPasswordRequest is the body for POST /auth/reset-password.
type resetPasswordRequest struct {
	Token       string `json:"token"`
	NewPassword string `json:"new_password"`
}

// changePasswordRequest is the body for PUT /auth/change-password.
type changePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// Register handles POST /auth/register.
func (ctrl *AuthController) Register(c *fiber.Ctx) error {
	ctx, span := tracer.Start(c.UserContext(), "AuthController.Register")
	defer span.End()

	log := middleware.LoggerFromLocals(c, ctrl.logger).With("handler", "AuthController.Register")

	var req registerRequest
	if err := c.BodyParser(&req); err != nil {
		domainErr := authdomain.InvalidRequestBody()
		log.Warn("failed to parse request body", "error", err.Error())
		observability.RecordError(span, domainErr)
		return domainErr
	}

	input := authusecases.RegisterInput{
		Name:      req.Name,
		Email:     req.Email,
		Password:  req.Password,
		DeviceID:  ctrl.resolveDeviceID(c, req.DeviceID),
		UserAgent: c.Get("User-Agent"),
		IPAddress: c.IP(),
	}

	output, err := ctrl.registerUC.Execute(ctx, input)
	if err != nil {
		observability.RecordError(span, err)
		return err
	}

	span.SetAttributes(attribute.Int64("user.id", output.User.ID))

	if ctrl.isCookieMode(c) {
		ctrl.setAuthCookies(c, output.TokenPair.AccessToken, output.TokenPair.RefreshToken)
		return c.Status(fiber.StatusCreated).JSON(fiber.Map{
			"user": output.User,
		})
	}

	return c.Status(fiber.StatusCreated).JSON(output)
}

// Login handles POST /auth/login.
func (ctrl *AuthController) Login(c *fiber.Ctx) error {
	ctx, span := tracer.Start(c.UserContext(), "AuthController.Login")
	defer span.End()

	log := middleware.LoggerFromLocals(c, ctrl.logger).With("handler", "AuthController.Login")

	var req loginRequest
	if err := c.BodyParser(&req); err != nil {
		domainErr := authdomain.InvalidRequestBody()
		log.Warn("failed to parse request body", "error", err.Error())
		observability.RecordError(span, domainErr)
		return domainErr
	}

	input := authusecases.LoginInput{
		Email:     req.Email,
		Password:  req.Password,
		DeviceID:  ctrl.resolveDeviceID(c, req.DeviceID),
		UserAgent: c.Get("User-Agent"),
		IPAddress: c.IP(),
	}

	output, err := ctrl.loginUC.Execute(ctx, input)
	if err != nil {
		observability.RecordError(span, err)
		return err
	}

	span.SetAttributes(attribute.Int64("user.id", output.User.ID))

	if ctrl.isCookieMode(c) {
		ctrl.setAuthCookies(c, output.TokenPair.AccessToken, output.TokenPair.RefreshToken)
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"user": output.User,
		})
	}

	return c.Status(fiber.StatusOK).JSON(output)
}

// RefreshToken handles POST /auth/refresh.
func (ctrl *AuthController) RefreshToken(c *fiber.Ctx) error {
	ctx, span := tracer.Start(c.UserContext(), "AuthController.RefreshToken")
	defer span.End()

	log := middleware.LoggerFromLocals(c, ctrl.logger).With("handler", "AuthController.RefreshToken")

	var req refreshRequest
	if err := c.BodyParser(&req); err != nil {
		domainErr := authdomain.InvalidRequestBody()
		log.Warn("failed to parse request body", "error", err.Error())
		observability.RecordError(span, domainErr)
		return domainErr
	}

	// Also try reading refresh_token from cookie if not in body
	refreshToken := req.RefreshToken
	if refreshToken == "" {
		refreshToken = c.Cookies("refresh_token")
	}

	input := authusecases.RefreshTokenInput{
		RefreshToken: refreshToken,
		DeviceID:     ctrl.resolveDeviceID(c, req.DeviceID),
		UserAgent:    c.Get("User-Agent"),
		IPAddress:    c.IP(),
	}

	output, err := ctrl.refreshTokenUC.Execute(ctx, input)
	if err != nil {
		observability.RecordError(span, err)
		return err
	}

	if ctrl.isCookieMode(c) {
		ctrl.setAuthCookies(c, output.TokenPair.AccessToken, output.TokenPair.RefreshToken)
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"message": "tokens refreshed",
		})
	}

	return c.Status(fiber.StatusOK).JSON(output)
}

// Logout handles POST /auth/logout.
func (ctrl *AuthController) Logout(c *fiber.Ctx) error {
	ctx, span := tracer.Start(c.UserContext(), "AuthController.Logout")
	defer span.End()

	log := middleware.LoggerFromLocals(c, ctrl.logger).With("handler", "AuthController.Logout")

	userID, ok := c.Locals("userID").(int64)
	if !ok || userID == 0 {
		domainErr := authdomain.MissingUserIdentity()
		log.Warn("userID not found in Locals")
		observability.RecordError(span, domainErr)
		return domainErr
	}

	tokenID, ok := c.Locals("tokenID").(string)
	if !ok || tokenID == "" {
		domainErr := authdomain.MissingTokenIdentity()
		log.Warn("tokenID not found in Locals")
		observability.RecordError(span, domainErr)
		return domainErr
	}

	span.SetAttributes(
		attribute.Int64("user.id", userID),
		attribute.String("token.id", tokenID),
	)

	// Extract the raw access token for blacklisting
	accessToken := extractRawToken(c)

	input := authusecases.LogoutInput{
		UserID:      userID,
		TokenID:     tokenID,
		AccessToken: accessToken,
	}

	if err := ctrl.logoutUC.Execute(ctx, input); err != nil {
		observability.RecordError(span, err)
		return err
	}

	// Clear auth cookies
	ctrl.clearAuthCookies(c)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "logged out successfully",
	})
}

// isCookieMode checks if the client requests cookie-based auth delivery.
func (ctrl *AuthController) isCookieMode(c *fiber.Ctx) bool {
	return c.Get("X-Auth-Mode") == "cookie"
}

// resolveDeviceID extracts the device ID from the request body field,
// X-Device-ID header, or defaults to "unknown".
func (ctrl *AuthController) resolveDeviceID(c *fiber.Ctx, bodyDeviceID string) string {
	if bodyDeviceID != "" {
		return bodyDeviceID
	}
	if headerDeviceID := c.Get("X-Device-ID"); headerDeviceID != "" {
		return headerDeviceID
	}
	return "unknown"
}

// setAuthCookies sets httpOnly secure cookies for access and refresh tokens.
func (ctrl *AuthController) setAuthCookies(c *fiber.Ctx, accessToken, refreshToken string) {
	accessMaxAge := ctrl.cfg.JWT.AccessExpiry * 60 // minutes to seconds

	c.Cookie(&fiber.Cookie{
		Name:     "access_token",
		Value:    accessToken,
		Path:     "/",
		MaxAge:   accessMaxAge,
		HTTPOnly: true,
		Secure:   true,
		SameSite: "Lax",
	})

	refreshMaxAge := ctrl.cfg.JWT.RefreshExpiry * 24 * 60 * 60 // days to seconds

	c.Cookie(&fiber.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		Path:     "/auth/refresh",
		MaxAge:   refreshMaxAge,
		HTTPOnly: true,
		Secure:   true,
		SameSite: "Lax",
	})
}

// clearAuthCookies expires auth cookies.
func (ctrl *AuthController) clearAuthCookies(c *fiber.Ctx) {
	c.Cookie(&fiber.Cookie{
		Name:     "access_token",
		Value:    "",
		Path:     "/",
		MaxAge:   0,
		Expires:  time.Now().Add(-1 * time.Hour),
		HTTPOnly: true,
		Secure:   true,
		SameSite: "Lax",
	})

	c.Cookie(&fiber.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/auth/refresh",
		MaxAge:   0,
		Expires:  time.Now().Add(-1 * time.Hour),
		HTTPOnly: true,
		Secure:   true,
		SameSite: "Lax",
	})
}

// ConfirmEmail handles POST /auth/confirm-email.
func (ctrl *AuthController) ConfirmEmail(c *fiber.Ctx) error {
	ctx, span := tracer.Start(c.UserContext(), "AuthController.ConfirmEmail")
	defer span.End()

	log := middleware.LoggerFromLocals(c, ctrl.logger).With("handler", "AuthController.ConfirmEmail")

	var req confirmEmailRequest
	_ = c.BodyParser(&req)
	if req.Token == "" {
		req.Token = c.Query("token")
	}

	if err := ctrl.confirmEmailUC.Execute(ctx, authusecases.ConfirmEmailInput{Token: req.Token}); err != nil {
		log.Warn("confirm email failed", "error", err.Error())
		observability.RecordError(span, err)
		return err
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "email confirmed"})
}

// ForgotPassword handles POST /auth/forgot-password.
func (ctrl *AuthController) ForgotPassword(c *fiber.Ctx) error {
	ctx, span := tracer.Start(c.UserContext(), "AuthController.ForgotPassword")
	defer span.End()

	log := middleware.LoggerFromLocals(c, ctrl.logger).With("handler", "AuthController.ForgotPassword")

	var req forgotPasswordRequest
	if err := c.BodyParser(&req); err != nil {
		domainErr := authdomain.InvalidRequestBody()
		log.Warn("failed to parse request body", "error", err.Error())
		observability.RecordError(span, domainErr)
		return domainErr
	}

	if err := ctrl.forgotPasswordUC.Execute(ctx, authusecases.ForgotPasswordInput{Email: req.Email}); err != nil {
		observability.RecordError(span, err)
		return err
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "if the email exists, a reset link has been sent",
	})
}

// ResetPassword handles POST /auth/reset-password.
func (ctrl *AuthController) ResetPassword(c *fiber.Ctx) error {
	ctx, span := tracer.Start(c.UserContext(), "AuthController.ResetPassword")
	defer span.End()

	log := middleware.LoggerFromLocals(c, ctrl.logger).With("handler", "AuthController.ResetPassword")

	var req resetPasswordRequest
	if err := c.BodyParser(&req); err != nil {
		domainErr := authdomain.InvalidRequestBody()
		log.Warn("failed to parse request body", "error", err.Error())
		observability.RecordError(span, domainErr)
		return domainErr
	}

	if err := ctrl.resetPasswordUC.Execute(ctx, authusecases.ResetPasswordInput{
		Token:       req.Token,
		NewPassword: req.NewPassword,
	}); err != nil {
		observability.RecordError(span, err)
		return err
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "password reset"})
}

// ChangePassword handles PUT /auth/change-password (authenticated).
func (ctrl *AuthController) ChangePassword(c *fiber.Ctx) error {
	ctx, span := tracer.Start(c.UserContext(), "AuthController.ChangePassword")
	defer span.End()

	log := middleware.LoggerFromLocals(c, ctrl.logger).With("handler", "AuthController.ChangePassword")

	userID, ok := c.Locals("userID").(int64)
	if !ok || userID == 0 {
		domainErr := authdomain.MissingUserIdentity()
		observability.RecordError(span, domainErr)
		return domainErr
	}

	var req changePasswordRequest
	if err := c.BodyParser(&req); err != nil {
		domainErr := authdomain.InvalidRequestBody()
		log.Warn("failed to parse request body", "error", err.Error())
		observability.RecordError(span, domainErr)
		return domainErr
	}

	if err := ctrl.changePasswordUC.Execute(ctx, authusecases.ChangePasswordInput{
		UserID:          userID,
		CurrentPassword: req.CurrentPassword,
		NewPassword:     req.NewPassword,
	}); err != nil {
		observability.RecordError(span, err)
		return err
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "password changed"})
}

// extractRawToken extracts the raw JWT string from the request for blacklisting.
func extractRawToken(c *fiber.Ctx) string {
	auth := c.Get("Authorization")
	if len(auth) > 7 && auth[:7] == "Bearer " {
		return auth[7:]
	}
	return c.Cookies("access_token")
}
