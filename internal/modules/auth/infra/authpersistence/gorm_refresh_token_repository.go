package authpersistence

import (
	"context"
	"errors"
	"time"

	"golang_boilerplate_module/internal/modules/auth/authdomain"
	"golang_boilerplate_module/internal/modules/auth/authdomain/authrepo"
	"golang_boilerplate_module/internal/shared/domain/exceptions"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"gorm.io/gorm"
)

var refreshTokenTracer = otel.Tracer("auth.refresh_token_repository")

// GORMRefreshTokenRepository implements authrepo.RefreshTokenRepository using GORM.
type GORMRefreshTokenRepository struct {
	db *gorm.DB
}

// NewGORMRefreshTokenRepository creates a new GORM-backed refresh token repository.
func NewGORMRefreshTokenRepository(db *gorm.DB) authrepo.RefreshTokenRepository {
	return &GORMRefreshTokenRepository{db: db}
}

func (r *GORMRefreshTokenRepository) Create(ctx context.Context, token *authdomain.RefreshToken) error {
	ctx, span := refreshTokenTracer.Start(ctx, "GORMRefreshTokenRepository.Create")
	defer span.End()

	span.SetAttributes(
		attribute.Int64("token.user_id", token.UserID),
		attribute.String("token.jti", token.JTI),
	)

	if err := r.db.WithContext(ctx).Create(token).Error; err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
		return exceptions.NewInternalException(map[string]any{"error": err.Error()})
	}

	span.SetStatus(codes.Ok, "created")
	return nil
}

func (r *GORMRefreshTokenRepository) GetByTokenHash(ctx context.Context, tokenHash string) (*authdomain.RefreshToken, error) {
	ctx, span := refreshTokenTracer.Start(ctx, "GORMRefreshTokenRepository.GetByTokenHash")
	defer span.End()

	span.SetAttributes(attribute.String("db.operation", "GetByTokenHash"))

	var token authdomain.RefreshToken
	err := r.db.WithContext(ctx).Where("token_hash = ?", tokenHash).First(&token).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		span.SetStatus(codes.Error, "not found")
		return nil, exceptions.NewNotFoundException("Refresh token not found", nil)
	}
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
		return nil, exceptions.NewInternalException(map[string]any{"error": err.Error()})
	}

	span.SetStatus(codes.Ok, "found")
	return &token, nil
}

func (r *GORMRefreshTokenRepository) GetByJTI(ctx context.Context, jti string) (*authdomain.RefreshToken, error) {
	ctx, span := refreshTokenTracer.Start(ctx, "GORMRefreshTokenRepository.GetByJTI")
	defer span.End()

	span.SetAttributes(attribute.String("db.operation", "GetByJTI"))

	var token authdomain.RefreshToken
	err := r.db.WithContext(ctx).Where("jti = ?", jti).First(&token).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		span.SetStatus(codes.Error, "not found")
		return nil, exceptions.NewNotFoundException("Refresh token not found", nil)
	}
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
		return nil, exceptions.NewInternalException(map[string]any{"error": err.Error()})
	}

	span.SetStatus(codes.Ok, "found")
	return &token, nil
}

func (r *GORMRefreshTokenRepository) MarkUsed(ctx context.Context, id uuid.UUID) error {
	ctx, span := refreshTokenTracer.Start(ctx, "GORMRefreshTokenRepository.MarkUsed")
	defer span.End()

	span.SetAttributes(attribute.String("token.id", id.String()))

	now := time.Now()
	result := r.db.WithContext(ctx).
		Model(&authdomain.RefreshToken{}).
		Where("id = ?", id).
		Updates(map[string]any{"used": true, "used_at": now})

	if result.Error != nil {
		span.SetStatus(codes.Error, result.Error.Error())
		span.RecordError(result.Error)
		return exceptions.NewInternalException(map[string]any{"error": result.Error.Error()})
	}

	span.SetStatus(codes.Ok, "marked used")
	return nil
}

func (r *GORMRefreshTokenRepository) RevokeByFamilyID(ctx context.Context, familyID uuid.UUID) error {
	ctx, span := refreshTokenTracer.Start(ctx, "GORMRefreshTokenRepository.RevokeByFamilyID")
	defer span.End()

	span.SetAttributes(attribute.String("token.family_id", familyID.String()))

	now := time.Now()
	result := r.db.WithContext(ctx).
		Model(&authdomain.RefreshToken{}).
		Where("family_id = ? AND revoked_at IS NULL", familyID).
		Update("revoked_at", now)

	if result.Error != nil {
		span.SetStatus(codes.Error, result.Error.Error())
		span.RecordError(result.Error)
		return exceptions.NewInternalException(map[string]any{"error": result.Error.Error()})
	}

	span.SetAttributes(attribute.Int64("tokens.revoked", result.RowsAffected))
	span.SetStatus(codes.Ok, "revoked by family")
	return nil
}

func (r *GORMRefreshTokenRepository) RevokeByUserID(ctx context.Context, userID int64) error {
	ctx, span := refreshTokenTracer.Start(ctx, "GORMRefreshTokenRepository.RevokeByUserID")
	defer span.End()

	span.SetAttributes(attribute.Int64("token.user_id", userID))

	now := time.Now()
	result := r.db.WithContext(ctx).
		Model(&authdomain.RefreshToken{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", now)

	if result.Error != nil {
		span.SetStatus(codes.Error, result.Error.Error())
		span.RecordError(result.Error)
		return exceptions.NewInternalException(map[string]any{"error": result.Error.Error()})
	}

	span.SetAttributes(attribute.Int64("tokens.revoked", result.RowsAffected))
	span.SetStatus(codes.Ok, "revoked by user")
	return nil
}

func (r *GORMRefreshTokenRepository) GetActiveByUserID(ctx context.Context, userID int64) ([]authdomain.RefreshToken, error) {
	ctx, span := refreshTokenTracer.Start(ctx, "GORMRefreshTokenRepository.GetActiveByUserID")
	defer span.End()

	span.SetAttributes(attribute.Int64("token.user_id", userID))

	var tokens []authdomain.RefreshToken
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND revoked_at IS NULL AND used = false AND expires_at > ?", userID, time.Now()).
		Find(&tokens).Error

	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
		return nil, exceptions.NewInternalException(map[string]any{"error": err.Error()})
	}

	span.SetAttributes(attribute.Int("tokens.count", len(tokens)))
	span.SetStatus(codes.Ok, "found")
	return tokens, nil
}

func (r *GORMRefreshTokenRepository) DeleteExpired(ctx context.Context) error {
	ctx, span := refreshTokenTracer.Start(ctx, "GORMRefreshTokenRepository.DeleteExpired")
	defer span.End()

	result := r.db.WithContext(ctx).
		Where("expires_at < ?", time.Now()).
		Delete(&authdomain.RefreshToken{})

	if result.Error != nil {
		span.SetStatus(codes.Error, result.Error.Error())
		span.RecordError(result.Error)
		return exceptions.NewInternalException(map[string]any{"error": result.Error.Error()})
	}

	span.SetAttributes(attribute.Int64("tokens.deleted", result.RowsAffected))
	span.SetStatus(codes.Ok, "expired tokens deleted")
	return nil
}
