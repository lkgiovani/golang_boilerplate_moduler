package plansusecases

import (
	"context"

	"golang_boilerplate_module/internal/modules/plans/plansdomain/plansrepo"
	"golang_boilerplate_module/internal/shared/domain/providers"
	"golang_boilerplate_module/internal/shared/infra/observability"

	"go.opentelemetry.io/otel/attribute"
)

// DeletePlanUseCase handles soft-deleting (deactivating) a subscription plan.
type DeletePlanUseCase struct {
	planRepo plansrepo.PlanRepository
	logger   providers.LoggerProvider
}

// NewDeletePlanUseCase creates a new DeletePlanUseCase with all required dependencies.
func NewDeletePlanUseCase(
	planRepo plansrepo.PlanRepository,
	logger providers.LoggerProvider,
) *DeletePlanUseCase {
	return &DeletePlanUseCase{
		planRepo: planRepo,
		logger:   logger,
	}
}

// Execute deactivates a plan by setting active to false (soft delete).
func (uc *DeletePlanUseCase) Execute(ctx context.Context, planID int64) error {
	ctx, span := plansTracer.Start(ctx, "DeletePlanUseCase.Execute")
	defer span.End()

	span.SetAttributes(attribute.Int64("plan.id", planID))

	log := observability.LoggerWithTrace(ctx, uc.logger).With("usecase", "DeletePlan", "planId", planID)

	_, err := uc.planRepo.UpdateByID(ctx, planID, map[string]any{"active": false})
	if err != nil {
		log.Error("failed to deactivate plan", "error", err.Error())
		observability.RecordError(span, err)
		return err
	}

	log.Info("plan deactivated successfully", "planId", planID)

	return nil
}
