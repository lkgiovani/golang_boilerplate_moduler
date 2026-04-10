package planshttp

import (
	"strconv"

	"golang_boilerplate_module/internal/modules/plans/application/plansusecases"
	"golang_boilerplate_module/internal/shared/domain/exceptions"
	"golang_boilerplate_module/internal/shared/domain/providers"
	"golang_boilerplate_module/internal/shared/infra/http/middleware"
	"golang_boilerplate_module/internal/shared/infra/observability"

	"github.com/gofiber/fiber/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

var tracer = otel.Tracer("plans.http")

// PlansController handles HTTP requests for plan management endpoints.
type PlansController struct {
	createPlanUC *plansusecases.CreatePlanUseCase
	updatePlanUC *plansusecases.UpdatePlanUseCase
	listPlansUC  *plansusecases.ListPlansUseCase
	getPlanUC    *plansusecases.GetPlanUseCase
	deletePlanUC *plansusecases.DeletePlanUseCase
	logger       providers.LoggerProvider
}

// NewPlansController creates a new PlansController with all required dependencies.
func NewPlansController(
	createPlanUC *plansusecases.CreatePlanUseCase,
	updatePlanUC *plansusecases.UpdatePlanUseCase,
	listPlansUC *plansusecases.ListPlansUseCase,
	getPlanUC *plansusecases.GetPlanUseCase,
	deletePlanUC *plansusecases.DeletePlanUseCase,
	logger providers.LoggerProvider,
) *PlansController {
	return &PlansController{
		createPlanUC: createPlanUC,
		updatePlanUC: updatePlanUC,
		listPlansUC:  listPlansUC,
		getPlanUC:    getPlanUC,
		deletePlanUC: deletePlanUC,
		logger:       logger,
	}
}

// CreatePlan handles POST /api/plans.
// Parses the request body into CreatePlanInput and delegates to CreatePlanUseCase.
// Returns 201 with the created plan JSON.
func (ctrl *PlansController) CreatePlan(c *fiber.Ctx) error {
	ctx, span := tracer.Start(c.UserContext(), "PlansController.CreatePlan")
	defer span.End()

	log := middleware.LoggerFromLocals(c, ctrl.logger).With("handler", "PlansController.CreatePlan")

	var input plansusecases.CreatePlanInput
	if err := c.BodyParser(&input); err != nil {
		domainErr := exceptions.NewBadRequestException("invalid request body", nil)
		log.Warn("failed to parse request body", "error", err.Error())
		observability.RecordError(span, domainErr)
		return domainErr
	}

	plan, err := ctrl.createPlanUC.Execute(ctx, input)
	if err != nil {
		observability.RecordError(span, err)
		return err
	}

	span.SetAttributes(attribute.Int64("plan.id", plan.ID))

	return c.Status(fiber.StatusCreated).JSON(plan)
}

// UpdatePlan handles PUT /api/plans/:id.
// Parses the plan ID from the URL and request body into UpdatePlanInput,
// then delegates to UpdatePlanUseCase. Returns 200 with the updated plan.
func (ctrl *PlansController) UpdatePlan(c *fiber.Ctx) error {
	ctx, span := tracer.Start(c.UserContext(), "PlansController.UpdatePlan")
	defer span.End()

	log := middleware.LoggerFromLocals(c, ctrl.logger).With("handler", "PlansController.UpdatePlan")

	planID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		domainErr := exceptions.NewBadRequestException("invalid plan ID", nil)
		log.Warn("failed to parse plan ID", "error", err.Error())
		observability.RecordError(span, domainErr)
		return domainErr
	}

	span.SetAttributes(attribute.Int64("plan.id", planID))

	var input plansusecases.UpdatePlanInput
	if err := c.BodyParser(&input); err != nil {
		domainErr := exceptions.NewBadRequestException("invalid request body", nil)
		log.Warn("failed to parse request body", "error", err.Error())
		observability.RecordError(span, domainErr)
		return domainErr
	}

	plan, err := ctrl.updatePlanUC.Execute(ctx, planID, input)
	if err != nil {
		observability.RecordError(span, err)
		return err
	}

	return c.Status(fiber.StatusOK).JSON(plan)
}

// ListPlans handles GET /api/plans.
// Returns 200 with a JSON array of all active plans.
func (ctrl *PlansController) ListPlans(c *fiber.Ctx) error {
	ctx, span := tracer.Start(c.UserContext(), "PlansController.ListPlans")
	defer span.End()

	plans, err := ctrl.listPlansUC.Execute(ctx)
	if err != nil {
		observability.RecordError(span, err)
		return err
	}

	return c.Status(fiber.StatusOK).JSON(plans)
}

// GetPlan handles GET /api/plans/:slug.
// Returns 200 with the plan matching the given slug.
func (ctrl *PlansController) GetPlan(c *fiber.Ctx) error {
	ctx, span := tracer.Start(c.UserContext(), "PlansController.GetPlan")
	defer span.End()

	slug := c.Params("slug")
	span.SetAttributes(attribute.String("plan.slug", slug))

	plan, err := ctrl.getPlanUC.Execute(ctx, slug)
	if err != nil {
		observability.RecordError(span, err)
		return err
	}

	return c.Status(fiber.StatusOK).JSON(plan)
}

// DeletePlan handles DELETE /api/plans/:id.
// Soft-deletes (deactivates) the plan and returns 204.
func (ctrl *PlansController) DeletePlan(c *fiber.Ctx) error {
	ctx, span := tracer.Start(c.UserContext(), "PlansController.DeletePlan")
	defer span.End()

	log := middleware.LoggerFromLocals(c, ctrl.logger).With("handler", "PlansController.DeletePlan")

	planID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		domainErr := exceptions.NewBadRequestException("invalid plan ID", nil)
		log.Warn("failed to parse plan ID", "error", err.Error())
		observability.RecordError(span, domainErr)
		return domainErr
	}

	span.SetAttributes(attribute.Int64("plan.id", planID))

	if err := ctrl.deletePlanUC.Execute(ctx, planID); err != nil {
		observability.RecordError(span, err)
		return err
	}

	return c.SendStatus(fiber.StatusNoContent)
}
