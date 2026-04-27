package plansusecases

import (
	"context"

	"golang_boilerplate_module/internal/modules/plans/plansdomain"
	"golang_boilerplate_module/internal/modules/plans/plansdomain/plansrepo"
	"golang_boilerplate_module/internal/shared/domain/providers"
	"golang_boilerplate_module/internal/shared/infra/observability"

	"go.opentelemetry.io/otel/attribute"
)

// UpdatePaymentMethodInput holds the user ID and new payment method token.
type UpdatePaymentMethodInput struct {
	UserID          int64  `json:"user_id"`
	PaymentMethodID string `json:"payment_method_id"`
}

// UpdatePaymentMethodUseCase replaces the default payment method on the user's active subscription.
type UpdatePaymentMethodUseCase struct {
	subRepo plansrepo.SubscriptionRepository
	gateway providers.PaymentGateway
	logger  providers.LoggerProvider
}

// NewUpdatePaymentMethodUseCase wires dependencies.
func NewUpdatePaymentMethodUseCase(
	subRepo plansrepo.SubscriptionRepository,
	gateway providers.PaymentGateway,
	logger providers.LoggerProvider,
) *UpdatePaymentMethodUseCase {
	return &UpdatePaymentMethodUseCase{subRepo: subRepo, gateway: gateway, logger: logger}
}

// Execute validates the active subscription and calls the gateway.
func (uc *UpdatePaymentMethodUseCase) Execute(ctx context.Context, input UpdatePaymentMethodInput) error {
	ctx, span := subscriptionTracer.Start(ctx, "UpdatePaymentMethodUseCase.Execute")
	defer span.End()
	span.SetAttributes(attribute.Int64("user.id", input.UserID))

	log := observability.LoggerWithTrace(ctx, uc.logger).With("usecase", "UpdatePaymentMethod", "userId", input.UserID)

	sub, err := uc.subRepo.GetActiveByUserID(ctx, input.UserID)
	if err != nil {
		observability.RecordError(span, err)
		return err
	}
	if sub.GatewayCustomerID == nil || *sub.GatewayCustomerID == "" {
		return plansdomain.CustomerNotFoundInGateway()
	}

	if err := uc.gateway.UpdatePaymentMethod(ctx, providers.UpdatePaymentMethodInput{
		CustomerID:      *sub.GatewayCustomerID,
		PaymentMethodID: input.PaymentMethodID,
	}); err != nil {
		log.Error("failed to update payment method", "error", err.Error())
		observability.RecordError(span, err)
		return plansdomain.FailedToUpdatePaymentMethod()
	}
	log.Info("payment method updated")
	return nil
}
