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
	GetByGatewaySubscriptionID(ctx context.Context, gatewayName, gatewaySubID string) (*plansdomain.Subscription, error)
	GetByGatewayCustomerID(ctx context.Context, gatewayName, gatewayCustID string) (*plansdomain.Subscription, error)
}
