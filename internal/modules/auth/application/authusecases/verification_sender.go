package authusecases

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"golang_boilerplate_module/internal/config"
	"golang_boilerplate_module/internal/modules/auth/authdomain"
	"golang_boilerplate_module/internal/modules/auth/authdomain/authrepo"
	"golang_boilerplate_module/internal/shared/domain/providers"
	"golang_boilerplate_module/internal/shared/infra/observability"

	"go.opentelemetry.io/otel/attribute"
)

const (
	emailVerificationTTL = 24 * time.Hour
	passwordResetTTL     = 1 * time.Hour
	tokenByteLength      = 32
)

// VerificationSender is a helper service that creates tokens in PostgreSQL and
// dispatches the corresponding email via the EmailProvider. It is reused by
// Register (initial confirmation) and ForgotPassword use cases.
type VerificationSender struct {
	verificationRepo authrepo.VerificationTokenRepository
	emailProvider    providers.EmailProvider
	cfg              *config.Config
	logger           providers.LoggerProvider
}

// NewVerificationSender creates a new VerificationSender.
func NewVerificationSender(
	verificationRepo authrepo.VerificationTokenRepository,
	emailProvider providers.EmailProvider,
	cfg *config.Config,
	logger providers.LoggerProvider,
) *VerificationSender {
	return &VerificationSender{
		verificationRepo: verificationRepo,
		emailProvider:    emailProvider,
		cfg:              cfg,
		logger:           logger,
	}
}

// SendEmailVerification generates a single-use token, persists it, and sends
// the confirmation email using the "confirmation" template.
func (s *VerificationSender) SendEmailVerification(ctx context.Context, userID int64, name, email string) error {
	ctx, span := authTracer.Start(ctx, "auth.send_email_verification")
	defer span.End()
	span.SetAttributes(attribute.Int64("user.id", userID), attribute.String("user.email", email))

	log := observability.LoggerWithTrace(ctx, s.logger).With("helper", "SendEmailVerification", "email", email)

	token, err := generateSecureToken()
	if err != nil {
		log.Error("failed to generate verification token", "error", err.Error())
		observability.RecordError(span, err)
		return authdomain.FailedToGenerateVerificationToken()
	}

	record := &authdomain.EmailVerificationToken{
		UserID:    userID,
		Email:     email,
		Token:     token,
		ExpiresAt: time.Now().Add(emailVerificationTTL),
		CreatedAt: time.Now(),
	}
	if err := s.verificationRepo.CreateEmailToken(ctx, record); err != nil {
		log.Error("failed to persist verification token", "error", err.Error())
		observability.RecordError(span, err)
		return err
	}

	confirmURL := fmt.Sprintf("%s/auth/confirm-email?token=%s", s.cfg.App.BaseURL, token)
	msg := providers.EmailMessage{
		To:           email,
		Subject:      "Confirm your email",
		TemplateName: "confirmation",
		Data: map[string]any{
			"Name":       name,
			"ConfirmURL": confirmURL,
		},
	}
	if err := s.emailProvider.Send(ctx, msg); err != nil {
		log.Error("failed to send confirmation email", "error", err.Error())
		observability.RecordError(span, err)
		return authdomain.FailedToSendVerificationEmail()
	}

	log.Info("verification email sent", "userId", userID)
	return nil
}

// SendPasswordReset generates a single-use token, persists it, and sends the
// password reset email using the "reset_password" template.
func (s *VerificationSender) SendPasswordReset(ctx context.Context, userID int64, name, email string) error {
	ctx, span := authTracer.Start(ctx, "auth.send_password_reset")
	defer span.End()
	span.SetAttributes(attribute.Int64("user.id", userID), attribute.String("user.email", email))

	log := observability.LoggerWithTrace(ctx, s.logger).With("helper", "SendPasswordReset", "email", email)

	token, err := generateSecureToken()
	if err != nil {
		log.Error("failed to generate reset token", "error", err.Error())
		observability.RecordError(span, err)
		return authdomain.FailedToGenerateResetToken()
	}

	record := &authdomain.PasswordResetToken{
		UserID:    userID,
		Email:     email,
		Token:     token,
		ExpiresAt: time.Now().Add(passwordResetTTL),
		CreatedAt: time.Now(),
	}
	if err := s.verificationRepo.CreatePasswordResetToken(ctx, record); err != nil {
		log.Error("failed to persist reset token", "error", err.Error())
		observability.RecordError(span, err)
		return err
	}

	resetURL := fmt.Sprintf("%s/auth/reset-password?token=%s", s.cfg.App.BaseURL, token)
	msg := providers.EmailMessage{
		To:           email,
		Subject:      "Reset your password",
		TemplateName: "reset_password",
		Data: map[string]any{
			"Name":     name,
			"ResetURL": resetURL,
		},
	}
	if err := s.emailProvider.Send(ctx, msg); err != nil {
		log.Error("failed to send reset email", "error", err.Error())
		observability.RecordError(span, err)
		return authdomain.FailedToSendResetEmail()
	}

	log.Info("password reset email sent", "userId", userID)
	return nil
}

// generateSecureToken returns a 64-char hex string backed by crypto/rand.
func generateSecureToken() (string, error) {
	b := make([]byte, tokenByteLength)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
