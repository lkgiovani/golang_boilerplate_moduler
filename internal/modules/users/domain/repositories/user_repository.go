package repositories

import (
	"context"

	"golang_boilerplate_module/internal/modules/users/domain"
	sharedrepo "golang_boilerplate_module/internal/shared/domain/repositories"
)

// UserRepository extends GenericRepository with user-specific queries.
type UserRepository interface {
	sharedrepo.GenericRepository[domain.User, uint]
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
}
