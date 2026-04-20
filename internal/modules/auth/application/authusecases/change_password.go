package authusecases

import (
	"context"

	"golang_boilerplate_module/internal/modules/auth/authdomain"
	"golang_boilerplate_module/internal/modules/users/usersdomain/usersrepo"
	"golang_boilerplate_module/internal/shared/domain/providers"
	"golang_boilerplate_module/internal/shared/infra/observability"

	"go.opentelemetry.io/otel/attribute"
	"golang.org/x/crypto/bcrypt"
)

// ChangePasswordInput holds the current and new passwords for an
// authenticated user.
type ChangePasswordInput struct {
	UserID          int64  `json:"-"`
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// ChangePasswordUseCase verifies the current password before updating it.
type ChangePasswordUseCase struct {
	userRepo usersrepo.UserRepository
	logger   providers.LoggerProvider
}

// NewChangePasswordUseCase creates a new ChangePasswordUseCase.
func NewChangePasswordUseCase(
	userRepo usersrepo.UserRepository,
	logger providers.LoggerProvider,
) *ChangePasswordUseCase {
	return &ChangePasswordUseCase{userRepo: userRepo, logger: logger}
}

// Execute verifies the current password and updates to the new one.
func (uc *ChangePasswordUseCase) Execute(ctx context.Context, input ChangePasswordInput) error {
	ctx, span := authTracer.Start(ctx, "auth.change_password")
	defer span.End()
	span.SetAttributes(attribute.Int64("user.id", input.UserID))

	log := observability.LoggerWithTrace(ctx, uc.logger).With("usecase", "ChangePassword", "userId", input.UserID)

	if input.CurrentPassword == "" || input.NewPassword == "" {
		err := authdomain.MissingPasswordFields()
		observability.RecordError(span, err)
		return err
	}
	if len(input.NewPassword) < 8 {
		err := authdomain.NewPasswordTooShort()
		observability.RecordError(span, err)
		return err
	}

	user, err := uc.userRepo.GetByID(ctx, input.UserID)
	if err != nil || user == nil {
		observability.RecordError(span, err)
		return authdomain.UserNotFound()
	}
	if user.Password == nil {
		err := authdomain.NoLocalPassword()
		observability.RecordError(span, err)
		return err
	}

	if bcrypt.CompareHashAndPassword([]byte(*user.Password), []byte(input.CurrentPassword)) != nil {
		err := authdomain.CurrentPasswordIncorrect()
		log.Warn("invalid current password")
		observability.RecordError(span, err)
		return err
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(input.NewPassword), 12)
	if err != nil {
		log.Error("failed to hash password", "error", err.Error())
		observability.RecordError(span, err)
		return authdomain.FailedToHashPassword()
	}

	if _, err := uc.userRepo.UpdateByID(ctx, input.UserID, map[string]any{
		"password": string(hashed),
	}); err != nil {
		log.Error("failed to update password", "error", err.Error())
		observability.RecordError(span, err)
		return err
	}

	log.Info("password changed")
	return nil
}
