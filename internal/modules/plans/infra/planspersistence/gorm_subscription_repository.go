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

// GetByStripeSubscriptionID retrieves a subscription by its Stripe subscription ID.
func (r *GORMSubscriptionRepository) GetByStripeSubscriptionID(ctx context.Context, stripeSubID string) (*plansdomain.Subscription, error) {
	ctx, span := dbTracer.Start(ctx, "GORMSubscriptionRepository.GetByStripeSubscriptionID")
	defer span.End()

	span.SetAttributes(attribute.String("db.operation", "GetByStripeSubscriptionID"))

	var sub plansdomain.Subscription
	err := r.db.WithContext(ctx).Where("stripe_subscription_id = ?", stripeSubID).First(&sub).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		span.SetStatus(codes.Error, "not found")
		return nil, plansdomain.SubscriptionNotFound()
	}
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
		return nil, wrapSubscriptionInternal(err, "GetByStripeSubscriptionID")
	}

	span.SetAttributes(attribute.Int64("subscription.id", sub.ID))
	return &sub, nil
}

// GetByStripeCustomerID retrieves a subscription by Stripe customer ID.
func (r *GORMSubscriptionRepository) GetByStripeCustomerID(ctx context.Context, stripeCustID string) (*plansdomain.Subscription, error) {
	ctx, span := dbTracer.Start(ctx, "GORMSubscriptionRepository.GetByStripeCustomerID")
	defer span.End()

	span.SetAttributes(attribute.String("db.operation", "GetByStripeCustomerID"))

	var sub plansdomain.Subscription
	err := r.db.WithContext(ctx).
		Where("stripe_customer_id = ? AND status IN ?", stripeCustID, []string{
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
		return nil, wrapSubscriptionInternal(err, "GetByStripeCustomerID")
	}

	span.SetAttributes(attribute.Int64("subscription.id", sub.ID))
	return &sub, nil
}
