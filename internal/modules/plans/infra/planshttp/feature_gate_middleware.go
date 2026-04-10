package planshttp

import (
	"golang_boilerplate_module/internal/modules/plans/plansdomain/plansrepo"
	"golang_boilerplate_module/internal/shared/domain/exceptions"
	"golang_boilerplate_module/internal/shared/domain/providers"

	"github.com/gofiber/fiber/v2"
)

// FeatureGateMiddleware checks whether the authenticated user has an active
// subscription whose plan includes the required feature.
type FeatureGateMiddleware struct {
	subRepo  plansrepo.SubscriptionRepository
	planRepo plansrepo.PlanRepository
	logger   providers.LoggerProvider
}

// NewFeatureGateMiddleware creates a new FeatureGateMiddleware with all required dependencies.
func NewFeatureGateMiddleware(
	subRepo plansrepo.SubscriptionRepository,
	planRepo plansrepo.PlanRepository,
	logger providers.LoggerProvider,
) *FeatureGateMiddleware {
	return &FeatureGateMiddleware{
		subRepo:  subRepo,
		planRepo: planRepo,
		logger:   logger,
	}
}

// RequireFeature returns a Fiber handler that blocks requests from users
// whose subscription plan does not include the specified feature key.
func (m *FeatureGateMiddleware) RequireFeature(feature string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID, ok := c.Locals("userID").(int64)
		if !ok {
			return exceptions.NewUnauthorizedException("authentication required", nil)
		}

		sub, err := m.subRepo.GetActiveByUserID(c.UserContext(), userID)
		if err != nil || sub == nil {
			return exceptions.NewForbiddenException("active subscription required", nil)
		}

		plan, err := m.planRepo.GetByID(c.UserContext(), sub.PlanID)
		if err != nil {
			return exceptions.NewInternalException(map[string]any{"error": "failed to load plan"})
		}

		if !plan.HasFeature(feature) {
			return exceptions.NewForbiddenException("plan does not include feature: "+feature, nil)
		}

		c.Locals("subscriptionID", sub.ID)
		c.Locals("planSlug", plan.Slug)
		return c.Next()
	}
}
