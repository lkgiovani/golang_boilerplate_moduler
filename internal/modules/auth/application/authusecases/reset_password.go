package authusecases

import (
	"context"
	"time"

	"golang_boilerplate_module/internal/modules/auth/authdomain/authrepo"
	"golang_boilerplate_module/internal/modules/users/usersdomain/usersrepo"
	"golang_boilerplate_module/internal/shared/domain/exceptions"
	"golang_boilerplate_module/internal/shared/domain/providers"
	"golang_boilerplate_module/internal/shared/infra/observability"

	"go.opentelemetry.io/otel/attribute"
	"golang.org/x/crypto/bcrypt"
)

// ResetPasswordInput holds the reset token and new password.
type ResetPasswordInput struct {
	Token       string `json:"token"`
	NewPassword string `json:"new_password"`
}

// ResetPasswordUseCase validates a password reset token and updates the
// user's password hash.
type ResetPasswordUseCase struct {
	verificationRepo authrepo.VerificationTokenRepository
	userRepo         usersrepo.UserRepository
	logger           providers.LoggerProvider
}

// NewResetPasswordUseCase creates a new ResetPasswordUseCase.
func NewResetPasswordUseCase(
	verificationRepo authrepo.VerificationTokenRepository,
	userRepo usersrepo.UserRepository,
	logger providers.LoggerProvider,
) *ResetPasswordUseCase {
	return &ResetPasswordUseCase{
		verificationRepo: verificationRepo,
		userRepo:         userRepo,
		logger:           logger,
	}
}

// Execute verifies the token and updates the user's password.
func (uc *ResetPasswordUseCase) Execute(ctx context.Context, input ResetPasswordInput) error {
	ctx, span := authTracer.Start(ctx, "auth.reset_password")
	defer span.End()

	log := observability.LoggerWithTrace(ctx, uc.logger).With("usecase", "ResetPassword")

	if input.Token == "" || input.NewPassword == "" {
		err := exceptions.NewBadRequestException("Token and new password are required", nil)
		observability.RecordError(span, err)
		return err
	}
	if len(input.NewPassword) < 8 {
		err := exceptions.NewBadRequestException("Password must be at least 8 characters", nil)
		observability.RecordError(span, err)
		return err
	}

	record, err := uc.verificationRepo.GetPasswordResetByToken(ctx, input.Token)
	if err != nil {
		log.Warn("reset token not found")
		observability.RecordError(span, err)
		return exceptions.NewNotFoundException("Invalid or expired token", nil)
	}

	if record.Used {
		err := exceptions.NewUnprocessableException("Token already used", nil)
		observability.RecordError(span, err)
		return err
	}
	if time.Now().After(record.ExpiresAt) {
		err := exceptions.NewUnprocessableException("Token has expired", nil)
		observability.RecordError(span, err)
		return err
	}

	span.SetAttributes(attribute.Int64("user.id", record.UserID))

	hashed, err := bcrypt.GenerateFromPassword([]byte(input.NewPassword), 12)
	if err != nil {
		log.Error("failed to hash password", "error", err.Error())
		observability.RecordError(span, err)
		return exceptions.NewInternalException(map[string]any{"error": "failed to hash password"})
	}

	if err := uc.verificationRepo.MarkPasswordResetUsed(ctx, record.ID); err != nil {
		log.Error("failed to mark reset token used", "error", err.Error())
		observability.RecordError(span, err)
		return err
	}

	if _, err := uc.userRepo.UpdateByID(ctx, record.UserID, map[string]any{
		"password": string(hashed),
	}); err != nil {
		log.Error("failed to update password", "error", err.Error())
		observability.RecordError(span, err)
		return err
	}

	log.Info("password reset successfully", "userId", record.UserID)
	return nil
}
