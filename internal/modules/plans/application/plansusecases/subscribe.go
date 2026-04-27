package plansusecases

import (
	"context"

	"golang_boilerplate_module/internal/config"
	"golang_boilerplate_module/internal/modules/plans/plansdomain"
	"golang_boilerplate_module/internal/modules/plans/plansdomain/plansrepo"
	"golang_boilerplate_module/internal/modules/users/usersdomain/usersrepo"
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

// SubscribeUseCase creates a gateway checkout session and persists a local subscription.
// Reuses User.GatewayCustomerID when present to avoid duplicate customers (D-09).
type SubscribeUseCase struct {
	planRepo    plansrepo.PlanRepository
	subRepo     plansrepo.SubscriptionRepository
	userRepo    usersrepo.UserRepository
	gateway     providers.PaymentGateway
	gatewayName string
	logger      providers.LoggerProvider
}

// NewSubscribeUseCase wires all dependencies via fx.
func NewSubscribeUseCase(
	cfg *config.Config,
	planRepo plansrepo.PlanRepository,
	subRepo plansrepo.SubscriptionRepository,
	userRepo usersrepo.UserRepository,
	gateway providers.PaymentGateway,
	logger providers.LoggerProvider,
) *SubscribeUseCase {
	return &SubscribeUseCase{
		planRepo:    planRepo,
		subRepo:     subRepo,
		userRepo:    userRepo,
		gateway:     gateway,
		gatewayName: cfg.PaymentGateway.Name,
		logger:      logger,
	}
}

// Execute runs the subscribe flow: validate → reuse-or-create gateway customer → checkout session → persist local subscription.
func (uc *SubscribeUseCase) Execute(ctx context.Context, input SubscribeInput) (*SubscribeOutput, error) {
	ctx, span := subscriptionTracer.Start(ctx, "SubscribeUseCase.Execute")
	defer span.End()

	log := observability.LoggerWithTrace(ctx, uc.logger).With(
		"usecase", "Subscribe",
		"userId", input.UserID,
		"planSlug", input.PlanSlug,
		"gateway", uc.gatewayName,
	)

	// 1. Reject duplicate active subscription
	existingSub, _ := uc.subRepo.GetActiveByUserID(ctx, input.UserID)
	if existingSub != nil {
		err := plansdomain.AlreadySubscribed()
		log.Warn("user already has an active subscription")
		observability.RecordError(span, err)
		return nil, err
	}

	// 2. Load plan
	plan, err := uc.planRepo.GetBySlug(ctx, input.PlanSlug)
	if err != nil {
		log.Error("failed to get plan", "error", err.Error())
		observability.RecordError(span, err)
		return nil, err
	}

	if plan.GatewayPriceID == nil || *plan.GatewayPriceID == "" {
		err := plansdomain.MissingGatewayPrice()
		log.Warn("plan has no gateway price configured", "planId", plan.ID)
		observability.RecordError(span, err)
		return nil, err
	}

	span.SetAttributes(
		attribute.Int64("plan.id", plan.ID),
		attribute.String("plan.slug", plan.Slug),
	)

	// 3. Reuse gateway customer if User already has one under the current gateway (D-09 fix)
	user, err := uc.userRepo.GetByID(ctx, input.UserID)
	if err != nil {
		log.Error("failed to load user", "error", err.Error())
		observability.RecordError(span, err)
		return nil, err
	}

	var customerID string
	if user.GatewayCustomerID != nil && *user.GatewayCustomerID != "" &&
		user.GatewayName != nil && *user.GatewayName == uc.gatewayName {
		customerID = *user.GatewayCustomerID
		span.SetAttributes(attribute.Bool("gateway.customer_reused", true))
	} else {
		customerID, err = uc.gateway.CreateCustomer(ctx, providers.CreateCustomerInput{
			Email: input.UserEmail,
			Name:  input.UserName,
		})
		if err != nil {
			log.Error("failed to create gateway customer", "error", err.Error())
			observability.RecordError(span, err)
			return nil, plansdomain.FailedToCreateCustomer()
		}
		if err := uc.userRepo.UpdateGatewayCustomer(ctx, user.ID, uc.gatewayName, customerID); err != nil {
			log.Error("failed to persist gateway customer on user", "error", err.Error())
			observability.RecordError(span, err)
			return nil, err
		}
		span.SetAttributes(attribute.Bool("gateway.customer_reused", false))
	}
	span.SetAttributes(attribute.String("gateway.customer_id", customerID))

	// 4. Checkout session
	result, err := uc.gateway.CreateCheckoutSession(ctx, providers.CreateCheckoutInput{
		CustomerID: customerID,
		PriceID:    *plan.GatewayPriceID,
		SuccessURL: input.SuccessURL,
		CancelURL:  input.CancelURL,
	})
	if err != nil {
		log.Error("failed to create checkout session", "error", err.Error())
		observability.RecordError(span, err)
		return nil, plansdomain.FailedToCreateCheckout()
	}

	// 5. Persist local subscription record
	subscription := &plansdomain.Subscription{
		UserID:            input.UserID,
		PlanID:            plan.ID,
		Status:            plansdomain.StatusIncomplete,
		GatewayCustomerID: &customerID,
		GatewayName:       uc.gatewayName,
	}
	if _, err := uc.subRepo.Add(ctx, subscription); err != nil {
		log.Error("failed to create subscription record", "error", err.Error())
		observability.RecordError(span, err)
		return nil, plansdomain.FailedToCreateSubscription()
	}

	log.Info("checkout session created",
		"sessionId", result.SessionID,
		"subscriptionId", subscription.ID)

	return &SubscribeOutput{
		SessionID:  result.SessionID,
		SessionURL: result.SessionURL,
	}, nil
}
