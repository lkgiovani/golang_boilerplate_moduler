package usershttp

import (
	"strconv"

	"golang_boilerplate_module/internal/modules/users/application/usersusecases"
	"golang_boilerplate_module/internal/shared/domain/exceptions"
	"golang_boilerplate_module/internal/shared/domain/providers"
	"golang_boilerplate_module/internal/shared/infra/http/middleware"
	"golang_boilerplate_module/internal/shared/infra/observability"

	"github.com/gofiber/fiber/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

var tracer = otel.Tracer("users.http")

type UserController struct {
	createUser *usersusecases.CreateUserUseCase
	getUser    *usersusecases.GetUserUseCase
	getMe      *usersusecases.GetMeUseCase
	updateMe   *usersusecases.UpdateMeUseCase
	logger     providers.LoggerProvider
}

func NewUserController(
	createUser *usersusecases.CreateUserUseCase,
	getUser *usersusecases.GetUserUseCase,
	getMe *usersusecases.GetMeUseCase,
	updateMe *usersusecases.UpdateMeUseCase,
	logger providers.LoggerProvider,
) *UserController {
	return &UserController{
		createUser: createUser,
		getUser:    getUser,
		getMe:      getMe,
		updateMe:   updateMe,
		logger:     logger,
	}
}

// updateMeRequest is the body for PUT /api/users/me.
type updateMeRequest struct {
	Name   *string `json:"name,omitempty"`
	ImgURL *string `json:"img_url,omitempty"`
}

// GetMe handles GET /api/users/me (authenticated).
func (ctrl *UserController) GetMe(c *fiber.Ctx) error {
	ctx, span := tracer.Start(c.UserContext(), "UserController.GetMe")
	defer span.End()

	userID, ok := c.Locals("userID").(int64)
	if !ok || userID == 0 {
		domainErr := exceptions.NewUnauthorizedException("missing user identity", nil)
		observability.RecordError(span, domainErr)
		return domainErr
	}
	span.SetAttributes(attribute.Int64("user.id", userID))

	user, err := ctrl.getMe.Execute(ctx, userID)
	if err != nil {
		observability.RecordError(span, err)
		return err
	}
	return c.JSON(user)
}

// UpdateMe handles PUT /api/users/me (authenticated).
func (ctrl *UserController) UpdateMe(c *fiber.Ctx) error {
	ctx, span := tracer.Start(c.UserContext(), "UserController.UpdateMe")
	defer span.End()

	log := middleware.LoggerFromLocals(c, ctrl.logger).With("handler", "UserController.UpdateMe")

	userID, ok := c.Locals("userID").(int64)
	if !ok || userID == 0 {
		domainErr := exceptions.NewUnauthorizedException("missing user identity", nil)
		observability.RecordError(span, domainErr)
		return domainErr
	}
	span.SetAttributes(attribute.Int64("user.id", userID))

	var req updateMeRequest
	if err := c.BodyParser(&req); err != nil {
		domainErr := exceptions.NewBadRequestException("Invalid request body", nil)
		log.Warn("failed to parse request body", "error", err.Error())
		observability.RecordError(span, domainErr)
		return domainErr
	}

	updated, err := ctrl.updateMe.Execute(ctx, usersusecases.UpdateMeInput{
		UserID: userID,
		Name:   req.Name,
		ImgURL: req.ImgURL,
	})
	if err != nil {
		observability.RecordError(span, err)
		return err
	}
	return c.JSON(updated)
}

func (ctrl *UserController) Create(c *fiber.Ctx) error {
	ctx, span := tracer.Start(c.UserContext(), "UserController.Create")
	defer span.End()

	log := middleware.LoggerFromLocals(c, ctrl.logger).With("handler", "UserController.Create")

	var input usersusecases.CreateUserInput
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

	span.SetAttributes(attribute.Int64("user.id", output.ID))
	return c.Status(fiber.StatusCreated).JSON(output)
}

func (ctrl *UserController) GetByID(c *fiber.Ctx) error {
	ctx, span := tracer.Start(c.UserContext(), "UserController.GetByID")
	defer span.End()

	log := middleware.LoggerFromLocals(c, ctrl.logger).With("handler", "UserController.GetByID")

	idStr := c.Params("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		domainErr := exceptions.NewBadRequestException("Invalid user ID", nil)
		log.Warn("invalid user id param", "id", idStr)
		observability.RecordError(span, domainErr)
		return domainErr
	}

	span.SetAttributes(attribute.Int64("user.id", id))

	output, err := ctrl.getUser.Execute(ctx, id)
	if err != nil {
		observability.RecordError(span, err)
		return err
	}

	return c.JSON(output)
}
