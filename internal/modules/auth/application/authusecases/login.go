package authusecases

import (
	"context"
	"time"

	"golang_boilerplate_module/internal/config"
	"golang_boilerplate_module/internal/modules/auth/authdomain"
	"golang_boilerplate_module/internal/modules/auth/authdomain/authprovider"
	"golang_boilerplate_module/internal/modules/auth/authdomain/authrepo"
	"golang_boilerplate_module/internal/modules/auth/infra/authproviders"
	"golang_boilerplate_module/internal/modules/users/usersdomain"
	"golang_boilerplate_module/internal/modules/users/usersdomain/usersrepo"
	"golang_boilerplate_module/internal/shared/domain/exceptions"
	"golang_boilerplate_module/internal/shared/domain/providers"
	"golang_boilerplate_module/internal/shared/infra/observability"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"golang.org/x/crypto/bcrypt"
)

// LoginInput holds the data required to authenticate a user.
type LoginInput struct {
	Email     string `json:"email"`
	Password  string `json:"password"`
	DeviceID  string `json:"device_id"`
	UserAgent string
	IPAddress string
}

// LoginOutput holds the result of a successful login.
type LoginOutput struct {
	User      usersdomain.User     `json:"user"`
	TokenPair authdomain.TokenPair `json:"token_pair"`
}

// LoginUseCase handles user authentication with credential verification,
// refresh token creation, and last access tracking.
type LoginUseCase struct {
	userRepo         usersrepo.UserRepository
	refreshTokenRepo authrepo.RefreshTokenRepository
	tokenProvider    authprovider.TokenProvider
	cfg              *config.Config
	logger           providers.LoggerProvider
}

// NewLoginUseCase creates a new LoginUseCase with all required dependencies.
func NewLoginUseCase(
	userRepo usersrepo.UserRepository,
	refreshTokenRepo authrepo.RefreshTokenRepository,
	tokenProvider authprovider.TokenProvider,
	cfg *config.Config,
	logger providers.LoggerProvider,
) *LoginUseCase {
	return &LoginUseCase{
		userRepo:         userRepo,
		refreshTokenRepo: refreshTokenRepo,
		tokenProvider:    tokenProvider,
		cfg:              cfg,
		logger:           logger,
	}
}

// Execute authenticates a user by email and password, generates a token pair,
// stores the refresh token in PostgreSQL, and updates the user's last access time.
func (uc *LoginUseCase) Execute(ctx context.Context, input LoginInput) (LoginOutput, error) {
	ctx, span := authTracer.Start(ctx, "auth.login")
	defer span.End()

	span.SetAttributes(attribute.String("user.email", input.Email))

	log := observability.LoggerWithTrace(ctx, uc.logger).With("usecase", "Login", "email", input.Email)

	// 1. Validate required fields
	if input.Email == "" || input.Password == "" {
		err := exceptions.NewBadRequestException("Email and password are required", nil)
		log.Warn("validation failed - missing required fields")
		observability.RecordError(span, err)
		return LoginOutput{}, err
	}

	// 2. Get user by email
	user, err := uc.userRepo.GetByEmail(ctx, input.Email)
	if err != nil || user == nil {
		err := exceptions.NewUnauthorizedException("invalid credentials", nil)
		log.Warn("user not found by email")
		observability.RecordError(span, err)
		return LoginOutput{}, err
	}

	span.SetAttributes(attribute.Int64("user.id", user.ID))

	// 3. Check account is active
	if !user.Active {
		err := exceptions.NewUnauthorizedException("account disabled", nil)
		log.Warn("login attempt on disabled account", "userId", user.ID)
		observability.RecordError(span, err)
		return LoginOutput{}, err
	}

	// 4. Check password is set (OAuth-only accounts cannot login with password)
	if user.Password == nil {
		err := exceptions.NewUnauthorizedException("invalid credentials", nil)
		log.Warn("login attempt on account without password (OAuth-only)", "userId", user.ID)
		observability.RecordError(span, err)
		return LoginOutput{}, err
	}

	// 5. Verify password with bcrypt
	if err := bcrypt.CompareHashAndPassword([]byte(*user.Password), []byte(input.Password)); err != nil {
		authErr := exceptions.NewUnauthorizedException("invalid credentials", nil)
		log.Warn("invalid password attempt", "userId", user.ID)
		observability.RecordError(span, authErr)
		return LoginOutput{}, authErr
	}

	// 6. Generate token pair
	tokenPair, err := uc.tokenProvider.GenerateTokenPair(authdomain.TokenClaims{
		UserID:        user.ID,
		Email:         user.Email,
		Admin:         user.Admin,
		EmailVerified: user.EmailVerified,
	})
	if err != nil {
		log.Error("failed to generate token pair", "error", err.Error())
		observability.RecordError(span, err)
		return LoginOutput{}, exceptions.NewInternalException(map[string]any{"error": "failed to generate tokens"})
	}

	// 7. Create refresh token in PG (new family for each login session)
	deviceID := input.DeviceID
	if deviceID == "" {
		deviceID = "unknown"
	}

	_, jti, err := uc.tokenProvider.ValidateRefreshToken(tokenPair.RefreshToken)
	if err != nil {
		log.Error("failed to extract refresh token JTI", "error", err.Error())
		observability.RecordError(span, err)
		return LoginOutput{}, exceptions.NewInternalException(map[string]any{"error": "failed to extract token claims"})
	}

	refreshToken := &authdomain.RefreshToken{
		ID:        uuid.New(),
		UserID:    user.ID,
		UserEmail: user.Email,
		DeviceID:  deviceID,
		JTI:       jti,
		FamilyID:  uuid.New(),
		TokenHash: authproviders.HashToken(tokenPair.RefreshToken),
		ExpiresAt: time.Now().Add(time.Duration(uc.cfg.JWT.RefreshExpiry) * 24 * time.Hour),
		CreatedAt: time.Now(),
		UserAgent: nilIfEmpty(input.UserAgent),
		IPAddress: nilIfEmpty(input.IPAddress),
	}

	if err := uc.refreshTokenRepo.Create(ctx, refreshToken); err != nil {
		log.Error("failed to store refresh token", "error", err.Error())
		observability.RecordError(span, err)
		return LoginOutput{}, err
	}

	// 8. Update last access time
	now := time.Now()
	if _, err := uc.userRepo.UpdateByID(ctx, user.ID, map[string]any{"LastAccess": now}); err != nil {
		log.Warn("failed to update last access time", "error", err.Error(), "userId", user.ID)
		// Non-critical: do not fail login if last_access update fails
	}

	log.Info("user logged in successfully", "userId", user.ID)

	return LoginOutput{
		User:      *user,
		TokenPair: *tokenPair,
	}, nil
}
