package planshttp

import (
	"golang_boilerplate_module/internal/config"
	"golang_boilerplate_module/internal/modules/plans/application/plansusecases"
	"golang_boilerplate_module/internal/modules/plans/plansdomain"
	"golang_boilerplate_module/internal/shared/domain/providers"
	"golang_boilerplate_module/internal/shared/infra/observability"

	"github.com/gofiber/fiber/v2"
	"go.opentelemetry.io/otel/attribute"
)

// WebhookController handles parametric /api/webhooks/:gateway POSTs.
// In single-active-gateway mode (D-11), the controller accepts a request only
// when :gateway matches the configured PAYMENT_GATEWAY. Any other value
// returns 404 — the adapter is not registered for that name.
type WebhookController struct {
	handleWebhookUC *plansusecases.HandleWebhookUseCase
	gateway         providers.PaymentGateway
	gatewayName     string
	logger          providers.LoggerProvider
}

// NewWebhookController wires dependencies.
func NewWebhookController(
	cfg *config.Config,
	handleWebhookUC *plansusecases.HandleWebhookUseCase,
	gateway providers.PaymentGateway,
	logger providers.LoggerProvider,
) *WebhookController {
	return &WebhookController{
		handleWebhookUC: handleWebhookUC,
		gateway:         gateway,
		gatewayName:     cfg.PaymentGateway.Name,
		logger:          logger,
	}
}

// HandleWebhook handles POST /api/webhooks/:gateway.
func (ctrl *WebhookController) HandleWebhook(c *fiber.Ctx) error {
	ctx, span := tracer.Start(c.UserContext(), "WebhookController.HandleWebhook")
	defer span.End()

	reqGateway := c.Params("gateway")
	span.SetAttributes(attribute.String("gateway.requested", reqGateway))

	if reqGateway != ctrl.gatewayName {
		return plansdomain.GatewayNotRegistered(reqGateway)
	}

	payload := c.Body()
	signature := c.Get(ctrl.gateway.SignatureHeader())

	if len(payload) == 0 {
		return plansdomain.EmptyWebhookPayload()
	}
	if signature == "" {
		return plansdomain.MissingWebhookSignature()
	}

	event, err := ctrl.gateway.ParseEvent(payload, signature)
	if err != nil {
		observability.RecordError(span, err)
		return plansdomain.InvalidWebhookSignature()
	}
	if event == nil {
		return plansdomain.InvalidGatewayEvent()
	}

	if err := ctrl.handleWebhookUC.Execute(ctx, plansusecases.HandleWebhookInput{
		GatewayName: ctrl.gatewayName,
		Event:       event,
	}); err != nil {
		observability.RecordError(span, err)
		return err
	}

	return c.SendStatus(fiber.StatusOK)
}
