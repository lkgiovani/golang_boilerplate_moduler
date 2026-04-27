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

// GetSubscriptionStatusOutput projects the gateway-reported state for an authenticated user.
type GetSubscriptionStatusOutput struct {
	Status             string     `json:"status"`
	CurrentPeriodStart *time.Time `json:"current_period_start,omitempty"`
	CurrentPeriodEnd   *time.Time `json:"current_period_end,omitempty"`
	CancelAtPeriodEnd  bool       `json:"cancel_at_period_end"`
}

// GetSubscriptionStatusUseCase fetches the live gateway status for the user's active subscription.
type GetSubscriptionStatusUseCase struct {
	subRepo plansrepo.SubscriptionRepository
	gateway providers.PaymentGateway
	logger  providers.LoggerProvider
}

// NewGetSubscriptionStatusUseCase wires dependencies.
func NewGetSubscriptionStatusUseCase(
	subRepo plansrepo.SubscriptionRepository,
	gateway providers.PaymentGateway,
	logger providers.LoggerProvider,
) *GetSubscriptionStatusUseCase {
	return &GetSubscriptionStatusUseCase{subRepo: subRepo, gateway: gateway, logger: logger}
}

// Execute loads the local subscription and projects the gateway status snapshot.
func (uc *GetSubscriptionStatusUseCase) Execute(ctx context.Context, userID int64) (*GetSubscriptionStatusOutput, error) {
	ctx, span := subscriptionTracer.Start(ctx, "GetSubscriptionStatusUseCase.Execute")
	defer span.End()
	span.SetAttributes(attribute.Int64("user.id", userID))

	log := observability.LoggerWithTrace(ctx, uc.logger).With("usecase", "GetSubscriptionStatus", "userId", userID)

	sub, err := uc.subRepo.GetActiveByUserID(ctx, userID)
	if err != nil {
		observability.RecordError(span, err)
		return nil, err
	}
	if sub.GatewaySubscriptionID == nil || *sub.GatewaySubscriptionID == "" {
		return nil, plansdomain.SubscriptionNotFound()
	}

	snap, err := uc.gateway.GetSubscriptionStatus(ctx, *sub.GatewaySubscriptionID)
	if err != nil {
		log.Error("failed to fetch subscription status from gateway", "error", err.Error())
		observability.RecordError(span, err)
		return nil, plansdomain.FailedToFetchSubscriptionStatus()
	}

	return &GetSubscriptionStatusOutput{
		Status:             snap.Status,
		CurrentPeriodStart: snap.CurrentPeriodStart,
		CurrentPeriodEnd:   snap.CurrentPeriodEnd,
		CancelAtPeriodEnd:  snap.CancelAtPeriodEnd,
	}, nil
}
