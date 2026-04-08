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
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"golang.org/x/crypto/bcrypt"
)

var authTracer = otel.Tracer("auth")

// RegisterInput holds the data required to register a new user.
type RegisterInput struct {
	Name      string `json:"name"`
	Email     string `json:"email"`
	Password  string `json:"password"`
	DeviceID  string `json:"device_id"`
	UserAgent string
	IPAddress string
}

// RegisterOutput holds the result of a successful registration.
type RegisterOutput struct {
	User      usersdomain.User      `json:"user"`
	TokenPair authdomain.TokenPair  `json:"token_pair"`
}

// RegisterUseCase handles new user registration with password hashing
// and initial refresh token creation.
type RegisterUseCase struct {
	userRepo         usersrepo.UserRepository
	refreshTokenRepo authrepo.RefreshTokenRepository
	tokenProvider    authprovider.TokenProvider
	cfg              *config.Config
	logger           providers.LoggerProvider
}

// NewRegisterUseCase creates a new RegisterUseCase with all required dependencies.
func NewRegisterUseCase(
	userRepo usersrepo.UserRepository,
	refreshTokenRepo authrepo.RefreshTokenRepository,
	tokenProvider authprovider.TokenProvider,
	cfg *config.Config,
	logger providers.LoggerProvider,
) *RegisterUseCase {
	return &RegisterUseCase{
		userRepo:         userRepo,
		refreshTokenRepo: refreshTokenRepo,
		tokenProvider:    tokenProvider,
		cfg:              cfg,
		logger:           logger,
	}
}

// Execute registers a new user, hashes the password with bcrypt, generates a
// token pair, and stores the refresh token in PostgreSQL with family-based tracking.
func (uc *RegisterUseCase) Execute(ctx context.Context, input RegisterInput) (RegisterOutput, error) {
	ctx, span := authTracer.Start(ctx, "auth.register")
	defer span.End()

	span.SetAttributes(attribute.String("user.email", input.Email))

	log := observability.LoggerWithTrace(ctx, uc.logger).With("usecase", "Register", "email", input.Email)

	// 1. Validate required fields
	if input.Name == "" || input.Email == "" || input.Password == "" {
		err := exceptions.NewBadRequestException("Name, email and password are required", nil)
		log.Warn("validation failed - missing required fields")
		observability.RecordError(span, err)
		return RegisterOutput{}, err
	}

	if len(input.Password) < 8 {
		err := exceptions.NewBadRequestException("Password must be at least 8 characters", nil)
		log.Warn("validation failed - password too short")
		observability.RecordError(span, err)
		return RegisterOutput{}, err
	}

	// 2. Check email uniqueness
	existing, _ := uc.userRepo.GetByEmail(ctx, input.Email)
	if existing != nil {
		err := exceptions.NewUnprocessableException(
			"Email already in use",
			map[string]any{"email": input.Email},
		)
		log.Warn("email already in use", "email", input.Email)
		observability.RecordError(span, err)
		return RegisterOutput{}, err
	}

	// 3. Hash password with bcrypt cost 12
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(input.Password), 12)
	if err != nil {
		log.Error("failed to hash password", "error", err.Error())
		observability.RecordError(span, err)
		return RegisterOutput{}, exceptions.NewInternalException(map[string]any{"error": "failed to hash password"})
	}
	hashedPassword := string(hashedBytes)

	// 4. Create user
	user := &usersdomain.User{
		Name:          input.Name,
		Email:         input.Email,
		Password:      &hashedPassword,
		Active:        true,
		Source:        "LOCAL",
		EmailVerified: false,
	}

	created, err := uc.userRepo.Add(ctx, user)
	if err != nil {
		log.Error("failed to create user", "error", err.Error())
		observability.RecordError(span, err)
		return RegisterOutput{}, err
	}

	span.SetAttributes(attribute.Int64("user.id", created.ID))

	// 5. Generate token pair
	tokenPair, err := uc.tokenProvider.GenerateTokenPair(authdomain.TokenClaims{
		UserID:        created.ID,
		Email:         created.Email,
		Admin:         created.Admin,
		EmailVerified: created.EmailVerified,
	})
	if err != nil {
		log.Error("failed to generate token pair", "error", err.Error())
		observability.RecordError(span, err)
		return RegisterOutput{}, exceptions.NewInternalException(map[string]any{"error": "failed to generate tokens"})
	}

	// 6. Create refresh token in PG with family tracking
	deviceID := input.DeviceID
	if deviceID == "" {
		deviceID = "unknown"
	}

	// Extract JTI from the refresh token
	_, jti, err := uc.tokenProvider.ValidateRefreshToken(tokenPair.RefreshToken)
	if err != nil {
		log.Error("failed to extract refresh token JTI", "error", err.Error())
		observability.RecordError(span, err)
		return RegisterOutput{}, exceptions.NewInternalException(map[string]any{"error": "failed to extract token claims"})
	}

	refreshToken := &authdomain.RefreshToken{
		ID:        uuid.New(),
		UserID:    created.ID,
		UserEmail: created.Email,
		DeviceID:  deviceID,
		JTI:       jti,
		FamilyID:  uuid.New(),
		TokenHash: authproviders.HashToken(tokenPair.RefreshToken),
		ExpiresAt: time.Now().Add(time.Duration(uc.cfg.JWT.RefreshExpiry) * 24 * time.Hour),
		CreatedAt: time.Now(),
		UserAgent: nilIfEmpty(input.UserAgent),
		IPAddress: nilIfEmpty(input.IPAddress),
	}

	// 7. Save refresh token
	if err := uc.refreshTokenRepo.Create(ctx, refreshToken); err != nil {
		log.Error("failed to store refresh token", "error", err.Error())
		observability.RecordError(span, err)
		return RegisterOutput{}, err
	}

	log.Info("user registered successfully", "userId", created.ID)

	return RegisterOutput{
		User:      *created,
		TokenPair: *tokenPair,
	}, nil
}

// nilIfEmpty returns a pointer to the string if non-empty, or nil otherwise.
func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
