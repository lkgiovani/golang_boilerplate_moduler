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
)

// ConfirmEmailInput holds the token submitted by the user.
type ConfirmEmailInput struct {
	Token string `json:"token"`
}

// ConfirmEmailUseCase validates a single-use email verification token and
// marks the user as email_verified=true.
type ConfirmEmailUseCase struct {
	verificationRepo authrepo.VerificationTokenRepository
	userRepo         usersrepo.UserRepository
	logger           providers.LoggerProvider
}

// NewConfirmEmailUseCase creates a new ConfirmEmailUseCase.
func NewConfirmEmailUseCase(
	verificationRepo authrepo.VerificationTokenRepository,
	userRepo usersrepo.UserRepository,
	logger providers.LoggerProvider,
) *ConfirmEmailUseCase {
	return &ConfirmEmailUseCase{
		verificationRepo: verificationRepo,
		userRepo:         userRepo,
		logger:           logger,
	}
}

// Execute validates the token and flips email_verified on the user.
func (uc *ConfirmEmailUseCase) Execute(ctx context.Context, input ConfirmEmailInput) error {
	ctx, span := authTracer.Start(ctx, "auth.confirm_email")
	defer span.End()

	log := observability.LoggerWithTrace(ctx, uc.logger).With("usecase", "ConfirmEmail")

	if input.Token == "" {
		err := exceptions.NewBadRequestException("Token is required", nil)
		observability.RecordError(span, err)
		return err
	}

	record, err := uc.verificationRepo.GetEmailTokenByToken(ctx, input.Token)
	if err != nil {
		log.Warn("verification token not found")
		observability.RecordError(span, err)
		return exceptions.NewNotFoundException("Invalid or expired token", nil)
	}

	if record.Used {
		err := exceptions.NewUnprocessableException("Token already used", nil)
		log.Warn("token already used", "tokenId", record.ID)
		observability.RecordError(span, err)
		return err
	}

	if time.Now().After(record.ExpiresAt) {
		err := exceptions.NewUnprocessableException("Token has expired", nil)
		log.Warn("token expired", "tokenId", record.ID)
		observability.RecordError(span, err)
		return err
	}

	span.SetAttributes(attribute.Int64("user.id", record.UserID))

	if err := uc.verificationRepo.MarkEmailTokenUsed(ctx, record.ID); err != nil {
		log.Error("failed to mark token used", "error", err.Error())
		observability.RecordError(span, err)
		return err
	}

	if _, err := uc.userRepo.UpdateByID(ctx, record.UserID, map[string]any{
		"email_verified": true,
	}); err != nil {
		log.Error("failed to mark user verified", "error", err.Error())
		observability.RecordError(span, err)
		return err
	}

	log.Info("email confirmed", "userId", record.UserID)
	return nil
}
