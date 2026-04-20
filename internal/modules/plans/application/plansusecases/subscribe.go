package plansusecases

import (
	"context"

	"golang_boilerplate_module/internal/modules/plans/plansdomain"
	"golang_boilerplate_module/internal/modules/plans/plansdomain/plansrepo"
	"golang_boilerplate_module/internal/shared/domain/providers"
	"golang_boilerplate_module/internal/shared/infra/observability"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

var subscriptionTracer = otel.Tracer("plans.usecases")

// SubscribeInput holds the data required to create a checkout session.
type SubscribeInput struct {
	UserID     int64  `json:"user_id"`
	UserEmail  string `json:"user_email"`
	UserName   string `json:"user_name"`
	PlanSlug   string `json:"plan_slug"`
	SuccessURL string `json:"success_url"`
	CancelURL  string `json:"cancel_url"`
}

// SubscribeOutput holds the result of creating a checkout session.
type SubscribeOutput struct {
	SessionID  string `json:"session_id"`
	SessionURL string `json:"session_url"`
}

// SubscribeUseCase handles creating a Stripe checkout session for a plan subscription.
type SubscribeUseCase struct {
	planRepo plansrepo.PlanRepository
	subRepo  plansrepo.SubscriptionRepository
	gateway  providers.PaymentGateway
	logger   providers.LoggerProvider
}

// NewSubscribeUseCase creates a new SubscribeUseCase with all required dependencies.
func NewSubscribeUseCase(
	planRepo plansrepo.PlanRepository,
	subRepo plansrepo.SubscriptionRepository,
	gateway providers.PaymentGateway,
	logger providers.LoggerProvider,
) *SubscribeUseCase {
	return &SubscribeUseCase{
		planRepo: planRepo,
		subRepo:  subRepo,
		gateway:  gateway,
		logger:   logger,
	}
}

// Execute creates a Stripe customer (lazily), initiates a checkout session,
// and stores a local subscription record with incomplete status.
func (uc *SubscribeUseCase) Execute(ctx context.Context, input SubscribeInput) (*SubscribeOutput, error) {
	ctx, span := subscriptionTracer.Start(ctx, "SubscribeUseCase.Execute")
	defer span.End()

	log := observability.LoggerWithTrace(ctx, uc.logger).With("usecase", "Subscribe", "userId", input.UserID, "planSlug", input.PlanSlug)

	// 1. Check if user already has an active subscription
	existingSub, _ := uc.subRepo.GetActiveByUserID(ctx, input.UserID)
	if existingSub != nil {
		err := plansdomain.AlreadySubscribed()
		log.Warn("user already has an active subscription")
		observability.RecordError(span, err)
		return nil, err
	}

	// 2. Get plan by slug
	plan, err := uc.planRepo.GetBySlug(ctx, input.PlanSlug)
	if err != nil {
		log.Error("failed to get plan", "error", err.Error())
		observability.RecordError(span, err)
		return nil, err
	}

	// 3. Check plan has StripePriceID
	if plan.StripePriceID == nil || *plan.StripePriceID == "" {
		err := plansdomain.MissingStripePrice()
		log.Warn("plan has no stripe price configured", "planId", plan.ID)
		observability.RecordError(span, err)
		return nil, err
	}

	span.SetAttributes(
		attribute.Int64("plan.id", plan.ID),
		attribute.String("plan.slug", plan.Slug),
	)

	// 4. Lazy Stripe customer creation
	customerID, err := uc.gateway.CreateCustomer(ctx, providers.CreateCustomerInput{
		Email: input.UserEmail,
		Name:  input.UserName,
	})
	if err != nil {
		log.Error("failed to create Stripe customer", "error", err.Error())
		observability.RecordError(span, err)
		return nil, plansdomain.FailedToCreateCustomer()
	}

	span.SetAttributes(attribute.String("stripe.customer_id", customerID))

	// 5. Create checkout session
	result, err := uc.gateway.CreateCheckoutSession(ctx, providers.CreateCheckoutInput{
		CustomerID: customerID,
		PriceID:    *plan.StripePriceID,
		SuccessURL: input.SuccessURL,
		CancelURL:  input.CancelURL,
	})
	if err != nil {
		log.Error("failed to create checkout session", "error", err.Error())
		observability.RecordError(span, err)
		return nil, plansdomain.FailedToCreateCheckout()
	}

	// 6. Create local subscription record with incomplete status
	subscription := &plansdomain.Subscription{
		UserID:           input.UserID,
		PlanID:           plan.ID,
		Status:           plansdomain.StatusIncomplete,
		StripeCustomerID: &customerID,
	}

	if _, err := uc.subRepo.Add(ctx, subscription); err != nil {
		log.Error("failed to create subscription record", "error", err.Error())
		observability.RecordError(span, err)
		return nil, plansdomain.FailedToCreateSubscription()
	}

	log.Info("checkout session created", "sessionId", result.SessionID, "subscriptionId", subscription.ID)

	return &SubscribeOutput{
		SessionID:  result.SessionID,
		SessionURL: result.SessionURL,
	}, nil
}
