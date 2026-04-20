package usersusecases

import (
	"context"

	"golang_boilerplate_module/internal/modules/users/usersdomain"
	"golang_boilerplate_module/internal/modules/users/usersdomain/usersrepo"
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
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type CreateUserUseCase struct {
	userRepo usersrepo.UserRepository
	logger   providers.LoggerProvider
}

func NewCreateUserUseCase(userRepo usersrepo.UserRepository, logger providers.LoggerProvider) *CreateUserUseCase {
	return &CreateUserUseCase{userRepo: userRepo, logger: logger}
}

func (uc *CreateUserUseCase) Execute(ctx context.Context, input CreateUserInput) (UserOutput, error) {
	ctx, span := userTracer.Start(ctx, "CreateUserUseCase.Execute")
	defer span.End()

	span.SetAttributes(attribute.String("user.email", input.Email))

	log := observability.LoggerWithTrace(ctx, uc.logger).With("usecase", "CreateUser", "email", input.Email)

	if input.Name == "" || input.Email == "" {
		err := usersdomain.MissingNameOrEmail()
		log.Warn("validation failed — name or email is empty")
		observability.RecordError(span, err)
		return UserOutput{}, err
	}

	existing, _ := uc.userRepo.GetByEmail(ctx, input.Email)
	if existing != nil {
		err := usersdomain.EmailTaken(input.Email)
		log.Warn("email already in use", "email", input.Email)
		observability.RecordError(span, err)
		return UserOutput{}, err
	}

	user := &usersdomain.User{
		Name:  input.Name,
		Email: input.Email,
	}

	created, err := uc.userRepo.Add(ctx, user)
	if err != nil {
		log.Error("failed to create user", "error", err.Error())
		observability.RecordError(span, err)
		return UserOutput{}, err
	}

	span.SetAttributes(attribute.Int64("user.id", created.ID))
	log.Info("user created successfully", "userId", created.ID)

	return UserOutput{
		ID:    created.ID,
		Name:  created.Name,
		Email: created.Email,
	}, nil
}
