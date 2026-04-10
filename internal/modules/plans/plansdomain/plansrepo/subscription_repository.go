package plansrepo

import (
	"context"

	"golang_boilerplate_module/internal/modules/plans/plansdomain"
	sharedrepo "golang_boilerplate_module/internal/shared/domain/repositories"
)

// SubscriptionRepository defines the contract for subscription data access.
type SubscriptionRepository interface {
	sharedrepo.GenericRepository[plansdomain.Subscription, int64]
	GetActiveByUserID(ctx context.Context, userID int64) (*plansdomain.Subscription, error)
	GetByStripeSubscriptionID(ctx context.Context, stripeSubID string) (*plansdomain.Subscription, error)
	GetByStripeCustomerID(ctx context.Context, stripeCustID string) (*plansdomain.Subscription, error)
}
