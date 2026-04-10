package plansrepo

import (
	"context"

	"golang_boilerplate_module/internal/modules/plans/plansdomain"
	sharedrepo "golang_boilerplate_module/internal/shared/domain/repositories"
)

// PlanRepository defines the contract for plan data access.
type PlanRepository interface {
	sharedrepo.GenericRepository[plansdomain.Plan, int64]
	GetBySlug(ctx context.Context, slug string) (*plansdomain.Plan, error)
	ListActive(ctx context.Context) ([]plansdomain.Plan, error)
}
