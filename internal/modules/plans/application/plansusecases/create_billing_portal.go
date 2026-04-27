package plansusecases

import (
	"context"

	"golang_boilerplate_module/internal/modules/plans/plansdomain"
	"golang_boilerplate_module/internal/modules/plans/plansdomain/plansrepo"
	"golang_boilerplate_module/internal/shared/domain/providers"
	"golang_boilerplate_module/internal/shared/infra/observability"

	"go.opentelemetry.io/otel/attribute"
)

// CreateBillingPortalInput holds the return URL for the portal session.
type CreateBillingPortalInput struct {
	UserID    int64  `json:"user_id"`
	ReturnURL string `json:"return_url"`
}

// CreateBillingPortalOutput holds the URL the client should redirect to.
type CreateBillingPortalOutput struct {
	URL string `json:"url"`
}

// CreateBillingPortalUseCase returns a gateway-hosted billing portal URL.
type CreateBillingPortalUseCase struct {
	subRepo plansrepo.SubscriptionRepository
	gateway providers.PaymentGateway
	logger  providers.LoggerProvider
}

// NewCreateBillingPortalUseCase wires dependencies.
func NewCreateBillingPortalUseCase(
	subRepo plansrepo.SubscriptionRepository,
	gateway providers.PaymentGateway,
	logger providers.LoggerProvider,
) *CreateBillingPortalUseCase {
	return &CreateBillingPortalUseCase{subRepo: subRepo, gateway: gateway, logger: logger}
}

// Execute loads the active subscription, validates gateway customer, creates portal session.
func (uc *CreateBillingPortalUseCase) Execute(ctx context.Context, input CreateBillingPortalInput) (*CreateBillingPortalOutput, error) {
	ctx, span := subscriptionTracer.Start(ctx, "CreateBillingPortalUseCase.Execute")
	defer span.End()
	span.SetAttributes(attribute.Int64("user.id", input.UserID))

	log := observability.LoggerWithTrace(ctx, uc.logger).With("usecase", "CreateBillingPortal", "userId", input.UserID)

	sub, err := uc.subRepo.GetActiveByUserID(ctx, input.UserID)
	if err != nil {
		observability.RecordError(span, err)
		return nil, err
	}
	if sub.GatewayCustomerID == nil || *sub.GatewayCustomerID == "" {
		return nil, plansdomain.CustomerNotFoundInGateway()
	}

	result, err := uc.gateway.CreateBillingPortalSession(ctx, providers.CreatePortalInput{
		CustomerID: *sub.GatewayCustomerID,
		ReturnURL:  input.ReturnURL,
	})
	if err != nil {
		log.Error("failed to create billing portal session", "error", err.Error())
		observability.RecordError(span, err)
		return nil, plansdomain.FailedToCreatePortalSession()
	}
	return &CreateBillingPortalOutput{URL: result.URL}, nil
}
