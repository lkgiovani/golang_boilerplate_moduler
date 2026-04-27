package plansusecases

import (
	"context"

	"golang_boilerplate_module/internal/modules/plans/plansdomain"
	"golang_boilerplate_module/internal/modules/plans/plansdomain/plansrepo"
	"golang_boilerplate_module/internal/shared/domain/providers"
	"golang_boilerplate_module/internal/shared/infra/observability"

	"go.opentelemetry.io/otel/attribute"
)

// RefundPaymentInput holds the refund request parameters.
type RefundPaymentInput struct {
	SubscriptionID int64  `json:"-"`
	ChargeID       string `json:"charge_id"`
	AmountCents    *int64 `json:"amount_cents,omitempty"`
	Reason         string `json:"reason,omitempty"`
}

// RefundPaymentOutput holds the gateway refund result.
type RefundPaymentOutput struct {
	RefundID string `json:"refund_id"`
	Status   string `json:"status"`
}

// RefundPaymentUseCase issues a refund against a prior charge.
type RefundPaymentUseCase struct {
	subRepo plansrepo.SubscriptionRepository
	gateway providers.PaymentGateway
	logger  providers.LoggerProvider
}

// NewRefundPaymentUseCase wires dependencies.
func NewRefundPaymentUseCase(
	subRepo plansrepo.SubscriptionRepository,
	gateway providers.PaymentGateway,
	logger providers.LoggerProvider,
) *RefundPaymentUseCase {
	return &RefundPaymentUseCase{subRepo: subRepo, gateway: gateway, logger: logger}
}

// Execute validates subscription existence and calls gateway.RefundPayment.
func (uc *RefundPaymentUseCase) Execute(ctx context.Context, input RefundPaymentInput) (*RefundPaymentOutput, error) {
	ctx, span := subscriptionTracer.Start(ctx, "RefundPaymentUseCase.Execute")
	defer span.End()
	span.SetAttributes(attribute.Int64("subscription.id", input.SubscriptionID))

	log := observability.LoggerWithTrace(ctx, uc.logger).With("usecase", "RefundPayment", "subscriptionId", input.SubscriptionID)

	if _, err := uc.subRepo.GetByID(ctx, input.SubscriptionID); err != nil {
		observability.RecordError(span, err)
		return nil, err
	}

	result, err := uc.gateway.RefundPayment(ctx, providers.RefundInput{
		ChargeID:    input.ChargeID,
		AmountCents: input.AmountCents,
		Reason:      input.Reason,
	})
	if err != nil {
		log.Error("refund failed", "error", err.Error())
		observability.RecordError(span, err)
		return nil, plansdomain.FailedToRefundPayment()
	}
	log.Info("refund issued", "refundId", result.RefundID)
	return &RefundPaymentOutput{RefundID: result.RefundID, Status: result.Status}, nil
}
