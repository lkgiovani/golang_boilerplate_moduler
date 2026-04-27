package usersrepo

import (
	"context"

	"golang_boilerplate_module/internal/modules/users/usersdomain"
	sharedrepo "golang_boilerplate_module/internal/shared/domain/repositories"
)

type UserRepository interface {
	sharedrepo.GenericRepository[usersdomain.User, int64]
	GetByEmail(ctx context.Context, email string) (*usersdomain.User, error)
	UpdateGatewayCustomer(ctx context.Context, userID int64, gatewayName, gatewayCustomerID string) error
}
