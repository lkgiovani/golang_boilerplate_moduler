package plansusecases

import (
	"context"
	"encoding/json"

	"golang_boilerplate_module/internal/modules/plans/plansdomain"
	"golang_boilerplate_module/internal/modules/plans/plansdomain/plansrepo"
	"golang_boilerplate_module/internal/shared/domain/exceptions"
	"golang_boilerplate_module/internal/shared/domain/providers"
	"golang_boilerplate_module/internal/shared/infra/observability"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

var plansTracer = otel.Tracer("plans.usecases")

// CreatePlanInput holds the data required to create a new plan.
type CreatePlanInput struct {
	Name            string           `json:"name"`
	Slug            string           `json:"slug"`
	Description     *string          `json:"description,omitempty"`
	PriceCents      int64            `json:"price_cents"`
	Currency        string           `json:"currency"`
	BillingInterval string           `json:"billing_interval"`
	Features        json.RawMessage  `json:"features,omitempty"`
	SortOrder       int              `json:"sort_order"`
	StripePriceID   *string          `json:"stripe_price_id,omitempty"`
}

// CreatePlanUseCase handles the creation of a new subscription plan.
type CreatePlanUseCase struct {
	planRepo plansrepo.PlanRepository
	logger   providers.LoggerProvider
}

// NewCreatePlanUseCase creates a new CreatePlanUseCase with all required dependencies.
func NewCreatePlanUseCase(
	planRepo plansrepo.PlanRepository,
	logger providers.LoggerProvider,
) *CreatePlanUseCase {
	return &CreatePlanUseCase{
		planRepo: planRepo,
		logger:   logger,
	}
}

// Execute creates a new plan after validating the input.
func (uc *CreatePlanUseCase) Execute(ctx context.Context, input CreatePlanInput) (*plansdomain.Plan, error) {
	ctx, span := plansTracer.Start(ctx, "CreatePlanUseCase.Execute")
	defer span.End()

	span.SetAttributes(attribute.String("plan.slug", input.Slug))

	log := observability.LoggerWithTrace(ctx, uc.logger).With("usecase", "CreatePlan", "slug", input.Slug)

	// Validate required fields
	if input.Name == "" {
		err := exceptions.NewBadRequestException("plan name is required", nil)
		log.Warn("validation failed - missing plan name")
		observability.RecordError(span, err)
		return nil, err
	}

	if input.Slug == "" {
		err := exceptions.NewBadRequestException("plan slug is required", nil)
		log.Warn("validation failed - missing plan slug")
		observability.RecordError(span, err)
		return nil, err
	}

	validIntervals := map[string]bool{"monthly": true, "yearly": true, "lifetime": true}
	if !validIntervals[input.BillingInterval] {
		err := exceptions.NewBadRequestException("billing_interval must be monthly, yearly, or lifetime", nil)
		log.Warn("validation failed - invalid billing interval", "interval", input.BillingInterval)
		observability.RecordError(span, err)
		return nil, err
	}

	// Apply defaults
	currency := input.Currency
	if currency == "" {
		currency = "BRL"
	}

	features := input.Features
	if features == nil {
		features = json.RawMessage("{}")
	}

	plan := &plansdomain.Plan{
		Name:            input.Name,
		Slug:            input.Slug,
		Description:     input.Description,
		PriceCents:      input.PriceCents,
		Currency:        currency,
		BillingInterval: input.BillingInterval,
		Features:        features,
		Active:          true,
		SortOrder:       input.SortOrder,
		StripePriceID:   input.StripePriceID,
	}

	created, err := uc.planRepo.Add(ctx, plan)
	if err != nil {
		log.Error("failed to create plan", "error", err.Error())
		observability.RecordError(span, err)
		return nil, err
	}

	span.SetAttributes(attribute.Int64("plan.id", created.ID))
	log.Info("plan created successfully", "planId", created.ID)

	return created, nil
}
