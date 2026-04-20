package authhttp

import (
	"strings"

	"golang_boilerplate_module/internal/modules/auth/authdomain"
	"golang_boilerplate_module/internal/modules/auth/authdomain/authprovider"
	"golang_boilerplate_module/internal/shared/domain/providers"

	"github.com/gofiber/fiber/v2"
)

// AuthMiddleware provides JWT-based authentication middleware with dual-mode
// token extraction (Authorization header + cookie) and Redis blacklist checking.
type AuthMiddleware struct {
	tokenProvider authprovider.TokenProvider
	cacheProvider providers.CacheProvider
	logger        providers.LoggerProvider
}

// NewAuthMiddleware creates a new AuthMiddleware with all required dependencies.
func NewAuthMiddleware(
	tp authprovider.TokenProvider,
	cp providers.CacheProvider,
	l providers.LoggerProvider,
) *AuthMiddleware {
	return &AuthMiddleware{
		tokenProvider: tp,
		cacheProvider: cp,
		logger:        l,
	}
}

// Required returns a Fiber handler that enforces authentication.
func (m *AuthMiddleware) Required() fiber.Handler {
	return func(c *fiber.Ctx) error {
		token := m.extractToken(c)
		if token == "" {
			return authdomain.MissingAuthToken()
		}

		claims, err := m.tokenProvider.ValidateAccessToken(token)
		if err != nil {
			m.logger.Warn("invalid access token", "error", err.Error())
			return authdomain.InvalidOrExpiredAuthToken()
		}

		// Check Redis blacklist
		blacklisted, err := m.cacheProvider.Exists(c.UserContext(), "blacklist:"+claims.TokenID)
		if err != nil {
			m.logger.Error("failed to check token blacklist", "error", err.Error(), "tokenId", claims.TokenID)
			return authdomain.FailedToCheckBlacklist()
		}
		if blacklisted {
			m.logger.Warn("blacklisted token used", "tokenId", claims.TokenID, "userId", claims.UserID)
			return authdomain.TokenRevoked()
		}

		// Set user claims in Locals for downstream handlers
		c.Locals("userID", claims.UserID)
		c.Locals("userEmail", claims.Email)
		c.Locals("userAdmin", claims.Admin)
		c.Locals("tokenID", claims.TokenID)

		return c.Next()
	}
}

// Optional returns a Fiber handler that attempts authentication but continues
// even if no token is found.
func (m *AuthMiddleware) Optional() fiber.Handler {
	return func(c *fiber.Ctx) error {
		token := m.extractToken(c)
		if token == "" {
			return c.Next()
		}

		claims, err := m.tokenProvider.ValidateAccessToken(token)
		if err != nil {
			return c.Next()
		}

		blacklisted, err := m.cacheProvider.Exists(c.UserContext(), "blacklist:"+claims.TokenID)
		if err != nil || blacklisted {
			return c.Next()
		}

		c.Locals("userID", claims.UserID)
		c.Locals("userEmail", claims.Email)
		c.Locals("userAdmin", claims.Admin)
		c.Locals("tokenID", claims.TokenID)

		return c.Next()
	}
}

// extractToken attempts to extract a JWT from the Authorization header first
// (Bearer scheme), falling back to the access_token cookie.
func (m *AuthMiddleware) extractToken(c *fiber.Ctx) string {
	auth := c.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	return c.Cookies("access_token")
}
