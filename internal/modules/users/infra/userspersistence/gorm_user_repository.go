package userspersistence

import (
	"context"
	"errors"

	"golang_boilerplate_module/internal/modules/users/usersdomain"
	"golang_boilerplate_module/internal/modules/users/usersdomain/usersrepo"
	"golang_boilerplate_module/internal/shared/domain/exceptions"
	sharedrepo "golang_boilerplate_module/internal/shared/infra/persistence/repositories"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"gorm.io/gorm"
)

var dbTracer = otel.Tracer("users.persistence")

type GORMUserRepository struct {
	*sharedrepo.GORMGenericRepository[usersdomain.User, uint]
	db *gorm.DB
}

func NewGORMUserRepository(db *gorm.DB) usersrepo.UserRepository {
	return &GORMUserRepository{
		GORMGenericRepository: sharedrepo.NewGORMGenericRepository[usersdomain.User, uint](db),
		db:                    db,
	}
}

func (r *GORMUserRepository) GetByEmail(ctx context.Context, email string) (*usersdomain.User, error) {
	ctx, span := dbTracer.Start(ctx, "GORMUserRepository.GetByEmail")
	defer span.End()

	span.SetAttributes(attribute.String("db.operation", "GetByEmail"))

	var user usersdomain.User
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		notFound := exceptions.NewNotFoundException("User not found", nil)
		span.SetStatus(codes.Error, "not found")
		return nil, notFound
	}
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
		return nil, exceptions.NewInternalException(map[string]any{"error": err.Error()})
	}

	span.SetAttributes(attribute.Int("user.id", int(user.ID)))
	return &user, nil
}
