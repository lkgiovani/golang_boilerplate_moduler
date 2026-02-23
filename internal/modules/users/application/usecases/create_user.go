package usecases

import (
	"context"

	"golang_boilerplate_module/internal/modules/users/domain"
	userrepo "golang_boilerplate_module/internal/modules/users/domain/repositories"
	"golang_boilerplate_module/internal/shared/domain/exceptions"
	"golang_boilerplate_module/internal/shared/domain/providers"
	"golang_boilerplate_module/internal/shared/infra/observability"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

var userTracer = otel.Tracer("users")

type CreateUserInput struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type UserOutput struct {
	ID    uint   `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// CreateUserUseCase handles user creation with email uniqueness validation.
type CreateUserUseCase struct {
	userRepo userrepo.UserRepository
	logger   providers.LoggerProvider
}

func NewCreateUserUseCase(userRepo userrepo.UserRepository, logger providers.LoggerProvider) *CreateUserUseCase {
	return &CreateUserUseCase{userRepo: userRepo, logger: logger}
}

func (uc *CreateUserUseCase) Execute(ctx context.Context, input CreateUserInput) (UserOutput, error) {
	ctx, span := userTracer.Start(ctx, "CreateUserUseCase.Execute")
	defer span.End()

	span.SetAttributes(attribute.String("user.email", input.Email))

	log := observability.LoggerWithTrace(ctx, uc.logger).With("usecase", "CreateUser", "email", input.Email)

	if input.Name == "" || input.Email == "" {
		err := exceptions.NewBadRequestException("Name and email are required", nil)
		log.Warn("validation failed — name or email is empty")
		observability.RecordError(span, err)
		return UserOutput{}, err
	}

	// Check email uniqueness
	existing, _ := uc.userRepo.GetByEmail(ctx, input.Email)
	if existing != nil {
		err := exceptions.NewUnprocessableException(
			"Email already in use",
			map[string]any{"email": input.Email},
		)
		log.Warn("email already in use", "email", input.Email)
		observability.RecordError(span, err)
		return UserOutput{}, err
	}

	user := &domain.User{
		Name:  input.Name,
		Email: input.Email,
	}

	created, err := uc.userRepo.Add(ctx, user)
	if err != nil {
		log.Error("failed to create user", "error", err.Error())
		observability.RecordError(span, err)
		return UserOutput{}, err
	}

	span.SetAttributes(attribute.Int("user.id", int(created.ID)))
	log.Info("user created successfully", "userId", created.ID)

	return UserOutput{
		ID:    created.ID,
		Name:  created.Name,
		Email: created.Email,
	}, nil
}
