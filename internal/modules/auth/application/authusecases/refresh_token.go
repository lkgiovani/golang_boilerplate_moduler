package authusecases

import (
	"context"
	"time"

	"golang_boilerplate_module/internal/config"
	"golang_boilerplate_module/internal/modules/auth/authdomain"
	"golang_boilerplate_module/internal/modules/auth/authdomain/authprovider"
	"golang_boilerplate_module/internal/modules/auth/authdomain/authrepo"
	"golang_boilerplate_module/internal/modules/auth/infra/authproviders"
	"golang_boilerplate_module/internal/modules/users/usersdomain/usersrepo"
	"golang_boilerplate_module/internal/shared/domain/providers"
	"golang_boilerplate_module/internal/shared/infra/observability"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
)

// RefreshTokenInput holds the data required to rotate a refresh token.
type RefreshTokenInput struct {
	RefreshToken string `json:"refresh_token"`
	DeviceID     string `json:"device_id"`
	UserAgent    string
	IPAddress    string
}

// RefreshTokenOutput holds the result of a successful token rotation.
type RefreshTokenOutput struct {
	TokenPair authdomain.TokenPair `json:"token_pair"`
}

// RefreshTokenUseCase handles refresh token rotation with family-based
// theft detection. If a previously-used token is replayed, the entire
// token family is revoked to protect the user.
type RefreshTokenUseCase struct {
	userRepo         usersrepo.UserRepository
	refreshTokenRepo authrepo.RefreshTokenRepository
	tokenProvider    authprovider.TokenProvider
	cfg              *config.Config
	logger           providers.LoggerProvider
}

// NewRefreshTokenUseCase creates a new RefreshTokenUseCase with all required dependencies.
func NewRefreshTokenUseCase(
	userRepo usersrepo.UserRepository,
	refreshTokenRepo authrepo.RefreshTokenRepository,
	tokenProvider authprovider.TokenProvider,
	cfg *config.Config,
	logger providers.LoggerProvider,
) *RefreshTokenUseCase {
	return &RefreshTokenUseCase{
		userRepo:         userRepo,
		refreshTokenRepo: refreshTokenRepo,
		tokenProvider:    tokenProvider,
		cfg:              cfg,
		logger:           logger,
	}
}

// Execute validates the incoming refresh token, performs theft detection,
// rotates the token (marking old as used), and returns a new token pair.
func (uc *RefreshTokenUseCase) Execute(ctx context.Context, input RefreshTokenInput) (RefreshTokenOutput, error) {
	ctx, span := authTracer.Start(ctx, "auth.refresh_token")
	defer span.End()

	log := observability.LoggerWithTrace(ctx, uc.logger).With("usecase", "RefreshToken")

	// 1. Validate refresh token JWT and extract claims
	if input.RefreshToken == "" {
		err := authdomain.MissingRefreshToken()
		log.Warn("validation failed - missing refresh token")
		observability.RecordError(span, err)
		return RefreshTokenOutput{}, err
	}

	userID, _, err := uc.tokenProvider.ValidateRefreshToken(input.RefreshToken)
	if err != nil {
		authErr := authdomain.InvalidRefreshToken()
		log.Warn("invalid refresh token JWT", "error", err.Error())
		observability.RecordError(span, authErr)
		return RefreshTokenOutput{}, authErr
	}

	span.SetAttributes(attribute.Int64("user.id", userID))

	// 2. Hash the token and look up in PG
	tokenHash := authproviders.HashToken(input.RefreshToken)
	storedToken, err := uc.refreshTokenRepo.GetByTokenHash(ctx, tokenHash)
	if err != nil || storedToken == nil {
		authErr := authdomain.RefreshTokenNotFound()
		log.Warn("refresh token not found in database")
		observability.RecordError(span, authErr)
		return RefreshTokenOutput{}, authErr
	}

	span.SetAttributes(attribute.String("token.family_id", storedToken.FamilyID.String()))

	// 3. Check if token is already revoked
	if storedToken.RevokedAt != nil {
		authErr := authdomain.RefreshTokenRevoked()
		log.Warn("attempt to use revoked refresh token", "tokenId", storedToken.ID.String())
		observability.RecordError(span, authErr)
		return RefreshTokenOutput{}, authErr
	}

	// 4. THEFT DETECTION: if token was already used, someone is replaying it
	if storedToken.Used {
		log.Warn("theft detected - refresh token replay attempt, revoking entire family",
			"tokenId", storedToken.ID.String(),
			"familyId", storedToken.FamilyID.String(),
			"userId", storedToken.UserID,
		)

		// Revoke the entire token family to protect the user
		if err := uc.refreshTokenRepo.RevokeByFamilyID(ctx, storedToken.FamilyID); err != nil {
			log.Error("failed to revoke token family after theft detection", "error", err.Error())
		}

		authErr := authdomain.TokenReuseDetected()
		observability.RecordError(span, authErr)
		return RefreshTokenOutput{}, authErr
	}

	// 5. Mark old token as used
	if err := uc.refreshTokenRepo.MarkUsed(ctx, storedToken.ID); err != nil {
		log.Error("failed to mark refresh token as used", "error", err.Error())
		observability.RecordError(span, err)
		return RefreshTokenOutput{}, authdomain.FailedToRotateToken()
	}

	// 6. Get user info for new token claims
	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil || user == nil {
		authErr := authdomain.UserNotFoundForAuth()
		log.Warn("user not found during token refresh", "userId", userID)
		observability.RecordError(span, authErr)
		return RefreshTokenOutput{}, authErr
	}

	// 7. Generate new token pair
	newTokenPair, err := uc.tokenProvider.GenerateTokenPair(authdomain.TokenClaims{
		UserID:        user.ID,
		Email:         user.Email,
		Admin:         user.Admin,
		EmailVerified: user.EmailVerified,
	})
	if err != nil {
		log.Error("failed to generate new token pair", "error", err.Error())
		observability.RecordError(span, err)
		return RefreshTokenOutput{}, authdomain.FailedToGenerateTokens()
	}

	// 8. Extract JTI from new refresh token
	_, newJTI, err := uc.tokenProvider.ValidateRefreshToken(newTokenPair.RefreshToken)
	if err != nil {
		log.Error("failed to extract new refresh token JTI", "error", err.Error())
		observability.RecordError(span, err)
		return RefreshTokenOutput{}, authdomain.FailedToExtractTokenClaims()
	}

	// 9. Create new refresh token in PG with same family, linked to old token
	deviceID := input.DeviceID
	if deviceID == "" {
		deviceID = storedToken.DeviceID
	}

	oldTokenID := storedToken.ID
	newRefreshToken := &authdomain.RefreshToken{
		ID:          uuid.New(),
		UserID:      user.ID,
		UserEmail:   user.Email,
		DeviceID:    deviceID,
		JTI:         newJTI,
		FamilyID:    storedToken.FamilyID, // Same family for rotation chain
		TokenHash:   authproviders.HashToken(newTokenPair.RefreshToken),
		ExpiresAt:   time.Now().Add(time.Duration(uc.cfg.JWT.RefreshExpiry) * 24 * time.Hour),
		CreatedAt:   time.Now(),
		RotatedFrom: &oldTokenID,
		UserAgent:   nilIfEmpty(input.UserAgent),
		IPAddress:   nilIfEmpty(input.IPAddress),
	}

	if err := uc.refreshTokenRepo.Create(ctx, newRefreshToken); err != nil {
		log.Error("failed to store rotated refresh token", "error", err.Error())
		observability.RecordError(span, err)
		return RefreshTokenOutput{}, err
	}

	log.Info("refresh token rotated successfully",
		"userId", user.ID,
		"familyId", storedToken.FamilyID.String(),
	)

	return RefreshTokenOutput{
		TokenPair: *newTokenPair,
	}, nil
}
