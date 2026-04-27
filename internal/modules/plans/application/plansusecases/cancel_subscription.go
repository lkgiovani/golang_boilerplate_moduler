package plansusecases

import (
	"context"
	"time"

	"golang_boilerplate_module/internal/modules/plans/plansdomain"
	"golang_boilerplate_module/internal/modules/plans/plansdomain/plansrepo"
	"golang_boilerplate_module/internal/shared/domain/providers"
	"golang_boilerplate_module/internal/shared/infra/observability"

	"go.opentelemetry.io/otel/attribute"
)

// CancelSubscriptionUseCase handles canceling a user's active subscription.
type CancelSubscriptionUseCase struct {
	subRepo plansrepo.SubscriptionRepository
	gateway providers.PaymentGateway
	logger  providers.LoggerProvider
}

// NewCancelSubscriptionUseCase creates a new CancelSubscriptionUseCase with all required dependencies.
func NewCancelSubscriptionUseCase(
	subRepo plansrepo.SubscriptionRepository,
	gateway providers.PaymentGateway,
	logger providers.LoggerProvider,
) *CancelSubscriptionUseCase {
	return &CancelSubscriptionUseCase{
		subRepo: subRepo,
		gateway: gateway,
		logger:  logger,
	}
}

// Execute cancels the user's active subscription both in Stripe and locally.
func (uc *CancelSubscriptionUseCase) Execute(ctx context.Context, userID int64) error {
	ctx, span := subscriptionTracer.Start(ctx, "CancelSubscriptionUseCase.Execute")
	defer span.End()

	span.SetAttributes(attribute.Int64("user.id", userID))

	log := observability.LoggerWithTrace(ctx, uc.logger).With("usecase", "CancelSubscription", "userId", userID)

	// 1. Get active subscription
	sub, err := uc.subRepo.GetActiveByUserID(ctx, userID)
	if err != nil {
		log.Error("failed to get active subscription", "error", err.Error())
		observability.RecordError(span, err)
		return err
	}
	if sub == nil {
		err := plansdomain.NoActiveSubscription()
		log.Warn("no active subscription found")
		observability.RecordError(span, err)
		return err
	}

	span.SetAttributes(attribute.Int64("subscription.id", sub.ID))

	// 2. Cancel on Stripe if stripe subscription ID exists
	if sub.GatewaySubscriptionID != nil {
		if err := uc.gateway.CancelSubscription(ctx, *sub.GatewaySubscriptionID); err != nil {
			// Log error but don't fail - subscription may already be canceled on gateway side
			log.Warn("failed to cancel gateway subscription", "gatewaySubscriptionId", *sub.GatewaySubscriptionID, "error", err.Error())
		}
	}

	// 3. Update local subscription status
	now := time.Now()
	if _, err := uc.subRepo.UpdateByID(ctx, sub.ID, map[string]any{
		"status":      string(plansdomain.StatusCanceled),
		"canceled_at": now,
	}); err != nil {
		log.Error("failed to update subscription status", "error", err.Error())
		observability.RecordError(span, err)
		return err
	}

	log.Info("subscription canceled", "subscriptionId", sub.ID)

	return nil
}
