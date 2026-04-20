package userspersistence

import (
	"context"
	"errors"

	"golang_boilerplate_module/internal/modules/users/usersdomain"
	"golang_boilerplate_module/internal/modules/users/usersdomain/usersrepo"
	"golang_boilerplate_module/internal/shared/domain/errs"
	sharedrepo "golang_boilerplate_module/internal/shared/infra/persistence/repositories"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"gorm.io/gorm"
)

var dbTracer = otel.Tracer("users.persistence")

type GORMUserRepository struct {
	*sharedrepo.GORMGenericRepository[usersdomain.User, int64]
	db *gorm.DB
}

func NewGORMUserRepository(db *gorm.DB) usersrepo.UserRepository {
	return &GORMUserRepository{
		GORMGenericRepository: sharedrepo.NewGORMGenericRepository[usersdomain.User, int64](db),
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
		span.SetStatus(codes.Error, "not found")
		return nil, usersdomain.UserNotFound()
	}
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
		wrapped := errs.Wrap(errs.EINTERNAL, err, "users_repository.GetByEmail failed")
		wrapped.Reportable = true
		return nil, wrapped
	}

	span.SetAttributes(attribute.Int64("user.id", user.ID))
	return &user, nil
}
