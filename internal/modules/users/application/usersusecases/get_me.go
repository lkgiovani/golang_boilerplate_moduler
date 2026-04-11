package usersusecases

import (
	"context"

	"golang_boilerplate_module/internal/modules/users/usersdomain"
	"golang_boilerplate_module/internal/modules/users/usersdomain/usersrepo"
	"golang_boilerplate_module/internal/shared/domain/exceptions"
	"golang_boilerplate_module/internal/shared/domain/providers"
	"golang_boilerplate_module/internal/shared/infra/observability"

	"go.opentelemetry.io/otel/attribute"
)

// GetMeUseCase returns the authenticated user's profile.
type GetMeUseCase struct {
	userRepo usersrepo.UserRepository
	logger   providers.LoggerProvider
}

// NewGetMeUseCase creates a new GetMeUseCase.
func NewGetMeUseCase(userRepo usersrepo.UserRepository, logger providers.LoggerProvider) *GetMeUseCase {
	return &GetMeUseCase{userRepo: userRepo, logger: logger}
}

// Execute fetches the user by ID.
func (uc *GetMeUseCase) Execute(ctx context.Context, userID int64) (*usersdomain.User, error) {
	ctx, span := userTracer.Start(ctx, "GetMeUseCase.Execute")
	defer span.End()
	span.SetAttributes(attribute.Int64("user.id", userID))

	log := observability.LoggerWithTrace(ctx, uc.logger).With("usecase", "GetMe", "userId", userID)

	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil || user == nil {
		log.Warn("user not found")
		observability.RecordError(span, err)
		return nil, exceptions.NewNotFoundException("User not found", nil)
	}
	return user, nil
}
