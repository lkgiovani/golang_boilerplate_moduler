package plansrepo

import (
	"context"

	"golang_boilerplate_module/internal/modules/plans/plansdomain"
	sharedrepo "golang_boilerplate_module/internal/shared/domain/repositories"
)

// PaymentEventRepository defines the contract for payment event data access.
type PaymentEventRepository interface {
	sharedrepo.GenericRepository[plansdomain.PaymentEvent, int64]
	GetByGatewayEventID(ctx context.Context, gatewayName, gatewayEventID string) (*plansdomain.PaymentEvent, error)
	MarkProcessed(ctx context.Context, id int64) error
	MarkFailed(ctx context.Context, id int64, errMsg string) error
}
