package planshttp

import (
	"strconv"

	"golang_boilerplate_module/internal/modules/plans/application/plansusecases"
	"golang_boilerplate_module/internal/modules/plans/plansdomain"
	"golang_boilerplate_module/internal/shared/domain/providers"
	"golang_boilerplate_module/internal/shared/infra/http/middleware"
	"golang_boilerplate_module/internal/shared/infra/observability"

	"github.com/gofiber/fiber/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

var tracer = otel.Tracer("plans.http")

// PlansController handles HTTP requests for plan management and subscription endpoints.
type PlansController struct {
	createPlanUC         *plansusecases.CreatePlanUseCase
	updatePlanUC         *plansusecases.UpdatePlanUseCase
	listPlansUC          *plansusecases.ListPlansUseCase
	getPlanUC            *plansusecases.GetPlanUseCase
	deletePlanUC         *plansusecases.DeletePlanUseCase
	subscribeUC          *plansusecases.SubscribeUseCase
	getSubscriptionUC    *plansusecases.GetSubscriptionUseCase
	cancelSubscriptionUC *plansusecases.CancelSubscriptionUseCase
	logger               providers.LoggerProvider
}

// NewPlansController creates a new PlansController with all required dependencies.
func NewPlansController(
	createPlanUC *plansusecases.CreatePlanUseCase,
	updatePlanUC *plansusecases.UpdatePlanUseCase,
	listPlansUC *plansusecases.ListPlansUseCase,
	getPlanUC *plansusecases.GetPlanUseCase,
	deletePlanUC *plansusecases.DeletePlanUseCase,
	subscribeUC *plansusecases.SubscribeUseCase,
	getSubscriptionUC *plansusecases.GetSubscriptionUseCase,
	cancelSubscriptionUC *plansusecases.CancelSubscriptionUseCase,
	logger providers.LoggerProvider,
) *PlansController {
	return &PlansController{
		createPlanUC:         createPlanUC,
		updatePlanUC:         updatePlanUC,
		listPlansUC:          listPlansUC,
		getPlanUC:            getPlanUC,
		deletePlanUC:         deletePlanUC,
		subscribeUC:          subscribeUC,
		getSubscriptionUC:    getSubscriptionUC,
		cancelSubscriptionUC: cancelSubscriptionUC,
		logger:               logger,
	}
}

// CreatePlan handles POST /api/plans.
func (ctrl *PlansController) CreatePlan(c *fiber.Ctx) error {
	ctx, span := tracer.Start(c.UserContext(), "PlansController.CreatePlan")
	defer span.End()

	log := middleware.LoggerFromLocals(c, ctrl.logger).With("handler", "PlansController.CreatePlan")

	var input plansusecases.CreatePlanInput
	if err := c.BodyParser(&input); err != nil {
		domainErr := plansdomain.InvalidRequestBody()
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
func (ctrl *PlansController) UpdatePlan(c *fiber.Ctx) error {
	ctx, span := tracer.Start(c.UserContext(), "PlansController.UpdatePlan")
	defer span.End()

	log := middleware.LoggerFromLocals(c, ctrl.logger).With("handler", "PlansController.UpdatePlan")

	planID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		domainErr := plansdomain.InvalidPlanID()
		log.Warn("failed to parse plan ID", "error", err.Error())
		observability.RecordError(span, domainErr)
		return domainErr
	}

	span.SetAttributes(attribute.Int64("plan.id", planID))

	var input plansusecases.UpdatePlanInput
	if err := c.BodyParser(&input); err != nil {
		domainErr := plansdomain.InvalidRequestBody()
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
func (ctrl *PlansController) DeletePlan(c *fiber.Ctx) error {
	ctx, span := tracer.Start(c.UserContext(), "PlansController.DeletePlan")
	defer span.End()

	log := middleware.LoggerFromLocals(c, ctrl.logger).With("handler", "PlansController.DeletePlan")

	planID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		domainErr := plansdomain.InvalidPlanID()
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

// subscribeRequest holds the JSON body for POST /api/subscriptions/checkout.
type subscribeRequest struct {
	PlanSlug   string `json:"plan_slug"`
	SuccessURL string `json:"success_url"`
	CancelURL  string `json:"cancel_url"`
}

// Subscribe handles POST /api/subscriptions/checkout.
func (ctrl *PlansController) Subscribe(c *fiber.Ctx) error {
	ctx, span := tracer.Start(c.UserContext(), "PlansController.Subscribe")
	defer span.End()

	log := middleware.LoggerFromLocals(c, ctrl.logger).With("handler", "PlansController.Subscribe")

	userID, ok := c.Locals("userID").(int64)
	if !ok {
		return plansdomain.AuthRequired()
	}
	userEmail, _ := c.Locals("userEmail").(string)

	var body subscribeRequest
	if err := c.BodyParser(&body); err != nil {
		domainErr := plansdomain.InvalidRequestBody()
		log.Warn("failed to parse request body", "error", err.Error())
		observability.RecordError(span, domainErr)
		return domainErr
	}

	input := plansusecases.SubscribeInput{
		UserID:     userID,
		UserEmail:  userEmail,
		PlanSlug:   body.PlanSlug,
		SuccessURL: body.SuccessURL,
		CancelURL:  body.CancelURL,
	}

	result, err := ctrl.subscribeUC.Execute(ctx, input)
	if err != nil {
		observability.RecordError(span, err)
		return err
	}

	return c.Status(fiber.StatusOK).JSON(result)
}

// GetSubscription handles GET /api/subscriptions/me.
func (ctrl *PlansController) GetSubscription(c *fiber.Ctx) error {
	ctx, span := tracer.Start(c.UserContext(), "PlansController.GetSubscription")
	defer span.End()

	userID, ok := c.Locals("userID").(int64)
	if !ok {
		return plansdomain.AuthRequired()
	}

	sub, err := ctrl.getSubscriptionUC.Execute(ctx, userID)
	if err != nil {
		observability.RecordError(span, err)
		return err
	}

	return c.Status(fiber.StatusOK).JSON(sub)
}

// CancelSubscription handles POST /api/subscriptions/cancel.
func (ctrl *PlansController) CancelSubscription(c *fiber.Ctx) error {
	ctx, span := tracer.Start(c.UserContext(), "PlansController.CancelSubscription")
	defer span.End()

	userID, ok := c.Locals("userID").(int64)
	if !ok {
		return plansdomain.AuthRequired()
	}

	if err := ctrl.cancelSubscriptionUC.Execute(ctx, userID); err != nil {
		observability.RecordError(span, err)
		return err
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "subscription canceled"})
}
