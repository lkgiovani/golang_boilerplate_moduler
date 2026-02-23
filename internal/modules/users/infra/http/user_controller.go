package http

import (
	"strconv"

	"golang_boilerplate_module/internal/modules/users/application/usecases"
	"golang_boilerplate_module/internal/shared/domain/exceptions"
	"golang_boilerplate_module/internal/shared/domain/providers"
	"golang_boilerplate_module/internal/shared/infra/http/middleware"
	"golang_boilerplate_module/internal/shared/infra/observability"

	"github.com/gofiber/fiber/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

var tracer = otel.Tracer("users.http")

// UserController handles user-related HTTP requests.
type UserController struct {
	createUser *usecases.CreateUserUseCase
	getUser    *usecases.GetUserUseCase
	logger     providers.LoggerProvider
}

func NewUserController(
	createUser *usecases.CreateUserUseCase,
	getUser *usecases.GetUserUseCase,
	logger providers.LoggerProvider,
) *UserController {
	return &UserController{
		createUser: createUser,
		getUser:    getUser,
		logger:     logger,
	}
}

// Create handles POST /api/users
func (ctrl *UserController) Create(c *fiber.Ctx) error {
	ctx, span := tracer.Start(c.UserContext(), "UserController.Create")
	defer span.End()

	log := middleware.LoggerFromLocals(c, ctrl.logger).With("handler", "UserController.Create")

	var input usecases.CreateUserInput
	if err := c.BodyParser(&input); err != nil {
		domainErr := exceptions.NewBadRequestException("Invalid request body", nil)
		log.Warn("failed to parse request body", "error", err.Error())
		observability.RecordError(span, domainErr)
		return domainErr
	}

	output, err := ctrl.createUser.Execute(ctx, input)
	if err != nil {
		observability.RecordError(span, err)
		return err
	}

	span.SetAttributes(attribute.Int("user.id", int(output.ID)))
	return c.Status(fiber.StatusCreated).JSON(output)
}

// GetByID handles GET /api/users/:id
func (ctrl *UserController) GetByID(c *fiber.Ctx) error {
	ctx, span := tracer.Start(c.UserContext(), "UserController.GetByID")
	defer span.End()

	log := middleware.LoggerFromLocals(c, ctrl.logger).With("handler", "UserController.GetByID")

	idStr := c.Params("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		domainErr := exceptions.NewBadRequestException("Invalid user ID", nil)
		log.Warn("invalid user id param", "id", idStr)
		observability.RecordError(span, domainErr)
		return domainErr
	}

	span.SetAttributes(attribute.Int("user.id", int(id)))

	output, err := ctrl.getUser.Execute(ctx, uint(id))
	if err != nil {
		observability.RecordError(span, err)
		return err
	}

	return c.JSON(output)
}
