package authusecases

import (
	"context"
	"fmt"
	"time"

	"golang_boilerplate_module/internal/config"
	"golang_boilerplate_module/internal/modules/auth/authdomain/authprovider"
	"golang_boilerplate_module/internal/modules/auth/authdomain/authrepo"
	"golang_boilerplate_module/internal/shared/domain/exceptions"
	"golang_boilerplate_module/internal/shared/domain/providers"
	"golang_boilerplate_module/internal/shared/infra/observability"

	"go.opentelemetry.io/otel/attribute"
)

// LogoutInput holds the data required to log out a user.
// UserID and TokenID come from the authenticated context (JWT claims).
type LogoutInput struct {
	UserID      int64
	TokenID     string
	AccessToken string
}

// LogoutUseCase handles user logout by blacklisting the access token in Redis
// and revoking all active refresh tokens in PostgreSQL.
type LogoutUseCase struct {
	refreshTokenRepo authrepo.RefreshTokenRepository
	tokenProvider    authprovider.TokenProvider
	cacheProvider    providers.CacheProvider
	cfg              *config.Config
	logger           providers.LoggerProvider
}

// NewLogoutUseCase creates a new LogoutUseCase with all required dependencies.
func NewLogoutUseCase(
	refreshTokenRepo authrepo.RefreshTokenRepository,
	tokenProvider authprovider.TokenProvider,
	cacheProvider providers.CacheProvider,
	cfg *config.Config,
	logger providers.LoggerProvider,
) *LogoutUseCase {
	return &LogoutUseCase{
		refreshTokenRepo: refreshTokenRepo,
		tokenProvider:    tokenProvider,
		cacheProvider:    cacheProvider,
		cfg:              cfg,
		logger:           logger,
	}
}

// Execute logs out the user by blacklisting the current access token in Redis
// (with TTL matching the token's remaining lifetime) and revoking all active
// refresh tokens for this user in PostgreSQL.
func (uc *LogoutUseCase) Execute(ctx context.Context, input LogoutInput) error {
	ctx, span := authTracer.Start(ctx, "auth.logout")
	defer span.End()

	span.SetAttributes(
		attribute.Int64("user.id", input.UserID),
		attribute.String("token.id", input.TokenID),
	)

	log := observability.LoggerWithTrace(ctx, uc.logger).With("usecase", "Logout", "userId", input.UserID)

	if input.UserID == 0 || input.TokenID == "" {
		err := exceptions.NewBadRequestException("User ID and token ID are required", nil)
		log.Warn("validation failed - missing user or token ID")
		observability.RecordError(span, err)
		return err
	}

	// 1. Blacklist the access token in Redis with remaining TTL
	blacklistKey := fmt.Sprintf("blacklist:%s", input.TokenID)
	remainingTTL := time.Duration(uc.cfg.JWT.AccessExpiry) * time.Minute

	if err := uc.cacheProvider.Set(ctx, blacklistKey, "1", remainingTTL); err != nil {
		log.Error("failed to blacklist access token in cache", "error", err.Error(), "tokenId", input.TokenID)
		observability.RecordError(span, err)
		return exceptions.NewInternalException(map[string]any{"error": "failed to blacklist access token"})
	}

	log.Debug("access token blacklisted", "tokenId", input.TokenID, "ttl", remainingTTL.String())

	// 2. Revoke all active refresh tokens for this user in PG
	if err := uc.refreshTokenRepo.RevokeByUserID(ctx, input.UserID); err != nil {
		log.Error("failed to revoke refresh tokens", "error", err.Error(), "userId", input.UserID)
		observability.RecordError(span, err)
		return err
	}

	log.Info("user logged out successfully", "userId", input.UserID)
	return nil
}
