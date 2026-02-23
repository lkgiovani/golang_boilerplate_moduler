package usersrepo

import (
	"context"

	"golang_boilerplate_module/internal/modules/users/usersdomain"
	sharedrepo "golang_boilerplate_module/internal/shared/domain/repositories"
)

type UserRepository interface {
	sharedrepo.GenericRepository[usersdomain.User, uint]
	GetByEmail(ctx context.Context, email string) (*usersdomain.User, error)
}
