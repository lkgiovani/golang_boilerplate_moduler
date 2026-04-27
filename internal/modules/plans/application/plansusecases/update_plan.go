package plansusecases

import (
	"context"
	"encoding/json"

	"golang_boilerplate_module/internal/modules/plans/plansdomain"
	"golang_boilerplate_module/internal/modules/plans/plansdomain/plansrepo"
	"golang_boilerplate_module/internal/shared/domain/providers"
	"golang_boilerplate_module/internal/shared/infra/observability"

	"go.opentelemetry.io/otel/attribute"
)

// UpdatePlanInput holds the optional fields that can be updated on a plan.
type UpdatePlanInput struct {
	Name            *string          `json:"name,omitempty"`
	Description     *string          `json:"description,omitempty"`
	PriceCents      *int64           `json:"price_cents,omitempty"`
	Currency        *string          `json:"currency,omitempty"`
	BillingInterval *string          `json:"billing_interval,omitempty"`
	Features        *json.RawMessage `json:"features,omitempty"`
	Active          *bool            `json:"active,omitempty"`
	SortOrder       *int             `json:"sort_order,omitempty"`
	GatewayPriceID  *string          `json:"gateway_price_id,omitempty"`
}

// UpdatePlanUseCase handles updating an existing subscription plan.
type UpdatePlanUseCase struct {
	planRepo plansrepo.PlanRepository
	logger   providers.LoggerProvider
}

// NewUpdatePlanUseCase creates a new UpdatePlanUseCase with all required dependencies.
func NewUpdatePlanUseCase(
	planRepo plansrepo.PlanRepository,
	logger providers.LoggerProvider,
) *UpdatePlanUseCase {
	return &UpdatePlanUseCase{
		planRepo: planRepo,
		logger:   logger,
	}
}

// Execute updates the plan identified by planID with the provided non-nil fields.
func (uc *UpdatePlanUseCase) Execute(ctx context.Context, planID int64, input UpdatePlanInput) (*plansdomain.Plan, error) {
	ctx, span := plansTracer.Start(ctx, "UpdatePlanUseCase.Execute")
	defer span.End()

	span.SetAttributes(attribute.Int64("plan.id", planID))

	log := observability.LoggerWithTrace(ctx, uc.logger).With("usecase", "UpdatePlan", "planId", planID)

	updates := make(map[string]any)

	if input.Name != nil {
		updates["name"] = *input.Name
	}
	if input.Description != nil {
		updates["description"] = *input.Description
	}
	if input.PriceCents != nil {
		updates["price_cents"] = *input.PriceCents
	}
	if input.Currency != nil {
		updates["currency"] = *input.Currency
	}
	if input.BillingInterval != nil {
		updates["billing_interval"] = *input.BillingInterval
	}
	if input.Features != nil {
		updates["features"] = *input.Features
	}
	if input.Active != nil {
		updates["active"] = *input.Active
	}
	if input.SortOrder != nil {
		updates["sort_order"] = *input.SortOrder
	}
	if input.GatewayPriceID != nil {
		updates["gateway_price_id"] = *input.GatewayPriceID
	}

	updated, err := uc.planRepo.UpdateByID(ctx, planID, updates)
	if err != nil {
		log.Error("failed to update plan", "error", err.Error())
		observability.RecordError(span, err)
		return nil, err
	}

	log.Info("plan updated successfully", "planId", planID)

	return updated, nil
}
