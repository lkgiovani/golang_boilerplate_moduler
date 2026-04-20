package authpersistence

import (
	"context"
	"errors"
	"time"

	"golang_boilerplate_module/internal/modules/auth/authdomain"
	"golang_boilerplate_module/internal/modules/auth/authdomain/authrepo"
	"golang_boilerplate_module/internal/shared/domain/errs"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"gorm.io/gorm"
)

var verificationTracer = otel.Tracer("auth.verification_repository")

// GORMVerificationTokenRepository implements authrepo.VerificationTokenRepository using GORM.
type GORMVerificationTokenRepository struct {
	db *gorm.DB
}

// NewGORMVerificationTokenRepository creates a new GORM-backed verification token repository.
func NewGORMVerificationTokenRepository(db *gorm.DB) authrepo.VerificationTokenRepository {
	return &GORMVerificationTokenRepository{db: db}
}

func wrapVerificationInternal(err error, op string) *errs.Error {
	e := errs.Wrap(errs.EINTERNAL, err, "verification_token_repository.%s failed", op)
	e.Reportable = true
	return e
}

func (r *GORMVerificationTokenRepository) CreateEmailToken(ctx context.Context, token *authdomain.EmailVerificationToken) error {
	ctx, span := verificationTracer.Start(ctx, "GORMVerificationTokenRepository.CreateEmailToken")
	defer span.End()

	span.SetAttributes(
		attribute.Int64("token.user_id", token.UserID),
		attribute.String("token.email", token.Email),
	)

	if err := r.db.WithContext(ctx).Create(token).Error; err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
		return wrapVerificationInternal(err, "CreateEmailToken")
	}

	span.SetStatus(codes.Ok, "created")
	return nil
}

func (r *GORMVerificationTokenRepository) GetEmailTokenByToken(ctx context.Context, token string) (*authdomain.EmailVerificationToken, error) {
	ctx, span := verificationTracer.Start(ctx, "GORMVerificationTokenRepository.GetEmailTokenByToken")
	defer span.End()

	span.SetAttributes(attribute.String("db.operation", "GetEmailTokenByToken"))

	var t authdomain.EmailVerificationToken
	err := r.db.WithContext(ctx).Where("token = ?", token).First(&t).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		span.SetStatus(codes.Error, "not found")
		return nil, authdomain.EmailVerificationTokenNotFound()
	}
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
		return nil, wrapVerificationInternal(err, "GetEmailTokenByToken")
	}

	span.SetStatus(codes.Ok, "found")
	return &t, nil
}

func (r *GORMVerificationTokenRepository) MarkEmailTokenUsed(ctx context.Context, id int64) error {
	ctx, span := verificationTracer.Start(ctx, "GORMVerificationTokenRepository.MarkEmailTokenUsed")
	defer span.End()

	span.SetAttributes(attribute.Int64("token.id", id))

	now := time.Now()
	result := r.db.WithContext(ctx).
		Model(&authdomain.EmailVerificationToken{}).
		Where("id = ?", id).
		Updates(map[string]any{"used": true, "verified_at": now})

	if result.Error != nil {
		span.SetStatus(codes.Error, result.Error.Error())
		span.RecordError(result.Error)
		return wrapVerificationInternal(result.Error, "MarkEmailTokenUsed")
	}

	span.SetStatus(codes.Ok, "marked used")
	return nil
}

func (r *GORMVerificationTokenRepository) CreatePasswordResetToken(ctx context.Context, token *authdomain.PasswordResetToken) error {
	ctx, span := verificationTracer.Start(ctx, "GORMVerificationTokenRepository.CreatePasswordResetToken")
	defer span.End()

	span.SetAttributes(
		attribute.Int64("token.user_id", token.UserID),
		attribute.String("token.email", token.Email),
	)

	if err := r.db.WithContext(ctx).Create(token).Error; err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
		return wrapVerificationInternal(err, "CreatePasswordResetToken")
	}

	span.SetStatus(codes.Ok, "created")
	return nil
}

func (r *GORMVerificationTokenRepository) GetPasswordResetByToken(ctx context.Context, token string) (*authdomain.PasswordResetToken, error) {
	ctx, span := verificationTracer.Start(ctx, "GORMVerificationTokenRepository.GetPasswordResetByToken")
	defer span.End()

	span.SetAttributes(attribute.String("db.operation", "GetPasswordResetByToken"))

	var t authdomain.PasswordResetToken
	err := r.db.WithContext(ctx).Where("token = ?", token).First(&t).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		span.SetStatus(codes.Error, "not found")
		return nil, authdomain.PasswordResetTokenNotFound()
	}
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
		return nil, wrapVerificationInternal(err, "GetPasswordResetByToken")
	}

	span.SetStatus(codes.Ok, "found")
	return &t, nil
}

func (r *GORMVerificationTokenRepository) MarkPasswordResetUsed(ctx context.Context, id int64) error {
	ctx, span := verificationTracer.Start(ctx, "GORMVerificationTokenRepository.MarkPasswordResetUsed")
	defer span.End()

	span.SetAttributes(attribute.Int64("token.id", id))

	now := time.Now()
	result := r.db.WithContext(ctx).
		Model(&authdomain.PasswordResetToken{}).
		Where("id = ?", id).
		Updates(map[string]any{"used": true, "used_at": now})

	if result.Error != nil {
		span.SetStatus(codes.Error, result.Error.Error())
		span.RecordError(result.Error)
		return wrapVerificationInternal(result.Error, "MarkPasswordResetUsed")
	}

	span.SetStatus(codes.Ok, "marked used")
	return nil
}
