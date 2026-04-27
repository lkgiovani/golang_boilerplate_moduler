package planspersistence

import (
	"context"
	"errors"

	"golang_boilerplate_module/internal/modules/plans/plansdomain"
	"golang_boilerplate_module/internal/modules/plans/plansdomain/plansrepo"
	"golang_boilerplate_module/internal/shared/domain/errs"
	sharedrepo "golang_boilerplate_module/internal/shared/infra/persistence/repositories"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"gorm.io/gorm"
)

// GORMSubscriptionRepository implements SubscriptionRepository using GORM.
type GORMSubscriptionRepository struct {
	*sharedrepo.GORMGenericRepository[plansdomain.Subscription, int64]
	db *gorm.DB
}

// NewGORMSubscriptionRepository creates a new GORM-based SubscriptionRepository.
func NewGORMSubscriptionRepository(db *gorm.DB) plansrepo.SubscriptionRepository {
	return &GORMSubscriptionRepository{
		GORMGenericRepository: sharedrepo.NewGORMGenericRepository[plansdomain.Subscription, int64](db),
		db:                    db,
	}
}

func wrapSubscriptionInternal(err error, op string) *errs.Error {
	e := errs.Wrap(errs.EINTERNAL, err, "subscription_repository.%s failed", op)
	e.Reportable = true
	return e
}

// GetActiveByUserID retrieves the active or trialing subscription for a user, preloading the Plan.
func (r *GORMSubscriptionRepository) GetActiveByUserID(ctx context.Context, userID int64) (*plansdomain.Subscription, error) {
	ctx, span := dbTracer.Start(ctx, "GORMSubscriptionRepository.GetActiveByUserID")
	defer span.End()

	span.SetAttributes(attribute.String("db.operation", "GetActiveByUserID"))

	var sub plansdomain.Subscription
	err := r.db.WithContext(ctx).
		Preload("Plan").
		Where("user_id = ? AND status IN ?", userID, []string{string(plansdomain.StatusActive), string(plansdomain.StatusTrialing)}).
		First(&sub).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		span.SetStatus(codes.Error, "not found")
		return nil, plansdomain.ActiveSubscriptionNotFound()
	}
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
		return nil, wrapSubscriptionInternal(err, "GetActiveByUserID")
	}

	span.SetAttributes(attribute.Int64("subscription.id", sub.ID))
	return &sub, nil
}

// GetByGatewaySubscriptionID retrieves a subscription by its gateway subscription ID.
func (r *GORMSubscriptionRepository) GetByGatewaySubscriptionID(ctx context.Context, gatewayName, gatewaySubID string) (*plansdomain.Subscription, error) {
	ctx, span := dbTracer.Start(ctx, "GORMSubscriptionRepository.GetByGatewaySubscriptionID")
	defer span.End()

	span.SetAttributes(
		attribute.String("db.operation", "GetByGatewaySubscriptionID"),
		attribute.String("gateway.name", gatewayName),
	)

	var sub plansdomain.Subscription
	err := r.db.WithContext(ctx).
		Where("gateway_name = ? AND gateway_subscription_id = ?", gatewayName, gatewaySubID).
		First(&sub).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		span.SetStatus(codes.Error, "not found")
		return nil, plansdomain.SubscriptionNotFound()
	}
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
		return nil, wrapSubscriptionInternal(err, "GetByGatewaySubscriptionID")
	}

	span.SetAttributes(attribute.Int64("subscription.id", sub.ID))
	return &sub, nil
}

// GetByGatewayCustomerID retrieves a subscription by gateway customer ID (scoped by gateway).
func (r *GORMSubscriptionRepository) GetByGatewayCustomerID(ctx context.Context, gatewayName, gatewayCustID string) (*plansdomain.Subscription, error) {
	ctx, span := dbTracer.Start(ctx, "GORMSubscriptionRepository.GetByGatewayCustomerID")
	defer span.End()

	span.SetAttributes(
		attribute.String("db.operation", "GetByGatewayCustomerID"),
		attribute.String("gateway.name", gatewayName),
	)

	var sub plansdomain.Subscription
	err := r.db.WithContext(ctx).
		Where("gateway_name = ? AND gateway_customer_id = ? AND status IN ?", gatewayName, gatewayCustID, []string{
			string(plansdomain.StatusActive),
			string(plansdomain.StatusTrialing),
			string(plansdomain.StatusPastDue),
			string(plansdomain.StatusIncomplete),
		}).
		First(&sub).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		span.SetStatus(codes.Error, "not found")
		return nil, plansdomain.SubscriptionNotFound()
	}
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
		return nil, wrapSubscriptionInternal(err, "GetByGatewayCustomerID")
	}

	span.SetAttributes(attribute.Int64("subscription.id", sub.ID))
	return &sub, nil
}
