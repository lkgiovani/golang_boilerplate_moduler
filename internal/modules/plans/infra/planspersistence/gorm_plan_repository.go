package planspersistence

import (
	"context"
	"errors"

	"golang_boilerplate_module/internal/modules/plans/plansdomain"
	"golang_boilerplate_module/internal/modules/plans/plansdomain/plansrepo"
	"golang_boilerplate_module/internal/shared/domain/errs"
	sharedrepo "golang_boilerplate_module/internal/shared/infra/persistence/repositories"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"gorm.io/gorm"
)

var dbTracer = otel.Tracer("plans.persistence")

// GORMPlanRepository implements PlanRepository using GORM.
type GORMPlanRepository struct {
	*sharedrepo.GORMGenericRepository[plansdomain.Plan, int64]
	db *gorm.DB
}

// NewGORMPlanRepository creates a new GORM-based PlanRepository.
func NewGORMPlanRepository(db *gorm.DB) plansrepo.PlanRepository {
	return &GORMPlanRepository{
		GORMGenericRepository: sharedrepo.NewGORMGenericRepository[plansdomain.Plan, int64](db),
		db:                    db,
	}
}

func wrapPlanInternal(err error, op string) *errs.Error {
	e := errs.Wrap(errs.EINTERNAL, err, "plan_repository.%s failed", op)
	e.Reportable = true
	return e
}

// GetBySlug retrieves a plan by its unique slug.
func (r *GORMPlanRepository) GetBySlug(ctx context.Context, slug string) (*plansdomain.Plan, error) {
	ctx, span := dbTracer.Start(ctx, "GORMPlanRepository.GetBySlug")
	defer span.End()

	span.SetAttributes(attribute.String("db.operation", "GetBySlug"))

	var plan plansdomain.Plan
	err := r.db.WithContext(ctx).Where("slug = ?", slug).First(&plan).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		span.SetStatus(codes.Error, "not found")
		return nil, plansdomain.PlanNotFound()
	}
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
		return nil, wrapPlanInternal(err, "GetBySlug")
	}

	span.SetAttributes(attribute.Int64("plan.id", plan.ID))
	return &plan, nil
}

// ListActive returns all active plans ordered by sort_order ascending.
func (r *GORMPlanRepository) ListActive(ctx context.Context) ([]plansdomain.Plan, error) {
	ctx, span := dbTracer.Start(ctx, "GORMPlanRepository.ListActive")
	defer span.End()

	span.SetAttributes(attribute.String("db.operation", "ListActive"))

	var plans []plansdomain.Plan
	err := r.db.WithContext(ctx).Where("active = true").Order("sort_order ASC").Find(&plans).Error
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
		return nil, wrapPlanInternal(err, "ListActive")
	}

	span.SetAttributes(attribute.Int("plan.count", len(plans)))
	return plans, nil
}
