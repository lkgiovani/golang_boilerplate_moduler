package planshttp

import (
	"golang_boilerplate_module/internal/modules/plans/application/plansusecases"
	"golang_boilerplate_module/internal/modules/plans/plansdomain"
	"golang_boilerplate_module/internal/shared/domain/providers"
	"golang_boilerplate_module/internal/shared/infra/observability"

	"github.com/gofiber/fiber/v2"
)

// WebhookController handles incoming Stripe webhook HTTP requests.
type WebhookController struct {
	handleWebhookUC *plansusecases.HandleWebhookUseCase
	logger          providers.LoggerProvider
}

// NewWebhookController creates a new WebhookController with all required dependencies.
func NewWebhookController(
	handleWebhookUC *plansusecases.HandleWebhookUseCase,
	logger providers.LoggerProvider,
) *WebhookController {
	return &WebhookController{
		handleWebhookUC: handleWebhookUC,
		logger:          logger,
	}
}

// HandleWebhook handles POST /api/webhooks/stripe.
func (ctrl *WebhookController) HandleWebhook(c *fiber.Ctx) error {
	ctx, span := tracer.Start(c.UserContext(), "WebhookController.HandleWebhook")
	defer span.End()

	// Read raw body FIRST - before any BodyParser call
	payload := c.Body()
	signature := c.Get("Stripe-Signature")

	if len(payload) == 0 {
		return plansdomain.EmptyWebhookPayload()
	}
	if signature == "" {
		return plansdomain.MissingStripeSignature()
	}

	input := plansusecases.HandleWebhookInput{
		Payload:   payload,
		Signature: signature,
	}

	if err := ctrl.handleWebhookUC.Execute(ctx, input); err != nil {
		observability.RecordError(span, err)
		return err
	}

	return c.SendStatus(fiber.StatusOK)
}
