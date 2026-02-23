package usersusecases

import (
	"context"

	"golang_boilerplate_module/internal/modules/users/usersdomain/usersrepo"
	"golang_boilerplate_module/internal/shared/domain/providers"
	"golang_boilerplate_module/internal/shared/infra/observability"

	"go.opentelemetry.io/otel/attribute"
)

type GetUserUseCase struct {
	userRepo usersrepo.UserRepository
	logger   providers.LoggerProvider
}

func NewGetUserUseCase(userRepo usersrepo.UserRepository, logger providers.LoggerProvider) *GetUserUseCase {
	return &GetUserUseCase{userRepo: userRepo, logger: logger}
}

func (uc *GetUserUseCase) Execute(ctx context.Context, id uint) (UserOutput, error) {
	ctx, span := userTracer.Start(ctx, "GetUserUseCase.Execute")
	defer span.End()

	span.SetAttributes(attribute.Int("user.id", int(id)))

	log := observability.LoggerWithTrace(ctx, uc.logger).With("usecase", "GetUser", "userId", id)

	user, err := uc.userRepo.GetByID(ctx, id)
	if err != nil {
		log.Warn("user not found", "userId", id)
		observability.RecordError(span, err)
		return UserOutput{}, err
	}

	log.Info("user retrieved", "userId", user.ID)

	return UserOutput{
		ID:    user.ID,
		Name:  user.Name,
		Email: user.Email,
	}, nil
}
