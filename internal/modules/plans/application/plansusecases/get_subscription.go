package plansusecases

import (
	"context"

	"golang_boilerplate_module/internal/modules/plans/plansdomain"
	"golang_boilerplate_module/internal/modules/plans/plansdomain/plansrepo"
	"golang_boilerplate_module/internal/shared/domain/exceptions"
	"golang_boilerplate_module/internal/shared/domain/providers"
	"golang_boilerplate_module/internal/shared/infra/observability"

	"go.opentelemetry.io/otel/attribute"
)

// GetSubscriptionUseCase handles retrieving a user's active subscription.
type GetSubscriptionUseCase struct {
	subRepo plansrepo.SubscriptionRepository
	logger  providers.LoggerProvider
}

// NewGetSubscriptionUseCase creates a new GetSubscriptionUseCase with all required dependencies.
func NewGetSubscriptionUseCase(
	subRepo plansrepo.SubscriptionRepository,
	logger providers.LoggerProvider,
) *GetSubscriptionUseCase {
	return &GetSubscriptionUseCase{
		subRepo: subRepo,
		logger:  logger,
	}
}

// Execute returns the user's active subscription with preloaded plan data.
func (uc *GetSubscriptionUseCase) Execute(ctx context.Context, userID int64) (*plansdomain.Subscription, error) {
	ctx, span := subscriptionTracer.Start(ctx, "GetSubscriptionUseCase.Execute")
	defer span.End()

	span.SetAttributes(attribute.Int64("user.id", userID))

	log := observability.LoggerWithTrace(ctx, uc.logger).With("usecase", "GetSubscription", "userId", userID)

	sub, err := uc.subRepo.GetActiveByUserID(ctx, userID)
	if err != nil {
		log.Error("failed to get active subscription", "error", err.Error())
		observability.RecordError(span, err)
		return nil, err
	}
	if sub == nil {
		err := exceptions.NewNotFoundException("no active subscription", nil)
		log.Warn("no active subscription found")
		observability.RecordError(span, err)
		return nil, err
	}

	span.SetAttributes(attribute.Int64("subscription.id", sub.ID))

	return sub, nil
}
