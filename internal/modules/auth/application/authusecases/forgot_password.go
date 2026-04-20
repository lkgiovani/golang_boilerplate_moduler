package authusecases

import (
	"context"

	"golang_boilerplate_module/internal/modules/auth/authdomain"
	"golang_boilerplate_module/internal/modules/users/usersdomain/usersrepo"
	"golang_boilerplate_module/internal/shared/domain/providers"
	"golang_boilerplate_module/internal/shared/infra/observability"

	"go.opentelemetry.io/otel/attribute"
)

// ForgotPasswordInput holds the email requesting a reset.
type ForgotPasswordInput struct {
	Email string `json:"email"`
}

// ForgotPasswordUseCase creates a password reset token and dispatches the
// email. To avoid user enumeration, missing users return nil without error.
type ForgotPasswordUseCase struct {
	userRepo usersrepo.UserRepository
	sender   *VerificationSender
	logger   providers.LoggerProvider
}

// NewForgotPasswordUseCase creates a new ForgotPasswordUseCase.
func NewForgotPasswordUseCase(
	userRepo usersrepo.UserRepository,
	sender *VerificationSender,
	logger providers.LoggerProvider,
) *ForgotPasswordUseCase {
	return &ForgotPasswordUseCase{
		userRepo: userRepo,
		sender:   sender,
		logger:   logger,
	}
}

// Execute looks up the user (silently) and dispatches a reset email if found.
func (uc *ForgotPasswordUseCase) Execute(ctx context.Context, input ForgotPasswordInput) error {
	ctx, span := authTracer.Start(ctx, "auth.forgot_password")
	defer span.End()
	span.SetAttributes(attribute.String("user.email", input.Email))

	log := observability.LoggerWithTrace(ctx, uc.logger).With("usecase", "ForgotPassword", "email", input.Email)

	if input.Email == "" {
		err := authdomain.MissingForgotEmail()
		observability.RecordError(span, err)
		return err
	}

	user, err := uc.userRepo.GetByEmail(ctx, input.Email)
	if err != nil || user == nil {
		// Silent success — do not leak account existence.
		log.Info("forgot password requested for unknown email")
		return nil
	}

	if err := uc.sender.SendPasswordReset(ctx, user.ID, user.Name, user.Email); err != nil {
		observability.RecordError(span, err)
		return err
	}
	return nil
}
