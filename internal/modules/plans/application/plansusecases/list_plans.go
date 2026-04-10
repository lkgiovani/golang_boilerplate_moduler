package plansusecases

import (
	"context"

	"golang_boilerplate_module/internal/modules/plans/plansdomain"
	"golang_boilerplate_module/internal/modules/plans/plansdomain/plansrepo"
	"golang_boilerplate_module/internal/shared/domain/providers"
	"golang_boilerplate_module/internal/shared/infra/observability"
)

// ListPlansUseCase handles listing all active subscription plans.
type ListPlansUseCase struct {
	planRepo plansrepo.PlanRepository
	logger   providers.LoggerProvider
}

// NewListPlansUseCase creates a new ListPlansUseCase with all required dependencies.
func NewListPlansUseCase(
	planRepo plansrepo.PlanRepository,
	logger providers.LoggerProvider,
) *ListPlansUseCase {
	return &ListPlansUseCase{
		planRepo: planRepo,
		logger:   logger,
	}
}

// Execute returns all active plans.
func (uc *ListPlansUseCase) Execute(ctx context.Context) ([]plansdomain.Plan, error) {
	ctx, span := plansTracer.Start(ctx, "ListPlansUseCase.Execute")
	defer span.End()

	log := observability.LoggerWithTrace(ctx, uc.logger).With("usecase", "ListPlans")

	plans, err := uc.planRepo.ListActive(ctx)
	if err != nil {
		log.Error("failed to list active plans", "error", err.Error())
		observability.RecordError(span, err)
		return nil, err
	}

	log.Info("listed active plans", "count", len(plans))

	return plans, nil
}
