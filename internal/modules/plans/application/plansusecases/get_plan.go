package plansusecases

import (
	"context"

	"golang_boilerplate_module/internal/modules/plans/plansdomain"
	"golang_boilerplate_module/internal/modules/plans/plansdomain/plansrepo"
	"golang_boilerplate_module/internal/shared/domain/providers"
	"golang_boilerplate_module/internal/shared/infra/observability"

	"go.opentelemetry.io/otel/attribute"
)

// GetPlanUseCase handles retrieving a single plan by its slug.
type GetPlanUseCase struct {
	planRepo plansrepo.PlanRepository
	logger   providers.LoggerProvider
}

// NewGetPlanUseCase creates a new GetPlanUseCase with all required dependencies.
func NewGetPlanUseCase(
	planRepo plansrepo.PlanRepository,
	logger providers.LoggerProvider,
) *GetPlanUseCase {
	return &GetPlanUseCase{
		planRepo: planRepo,
		logger:   logger,
	}
}

// Execute retrieves a plan by its slug.
func (uc *GetPlanUseCase) Execute(ctx context.Context, slug string) (*plansdomain.Plan, error) {
	ctx, span := plansTracer.Start(ctx, "GetPlanUseCase.Execute")
	defer span.End()

	span.SetAttributes(attribute.String("plan.slug", slug))

	log := observability.LoggerWithTrace(ctx, uc.logger).With("usecase", "GetPlan", "slug", slug)

	plan, err := uc.planRepo.GetBySlug(ctx, slug)
	if err != nil {
		log.Error("failed to get plan by slug", "error", err.Error())
		observability.RecordError(span, err)
		return nil, err
	}

	log.Info("plan retrieved successfully", "planId", plan.ID)

	return plan, nil
}
