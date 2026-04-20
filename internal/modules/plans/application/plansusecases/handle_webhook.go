package plansusecases

import (
	"context"
	"encoding/json"
	"time"

	"golang_boilerplate_module/internal/modules/plans/plansdomain"
	"golang_boilerplate_module/internal/modules/plans/plansdomain/plansrepo"
	"golang_boilerplate_module/internal/shared/domain/providers"
	"golang_boilerplate_module/internal/shared/infra/observability"

	"go.opentelemetry.io/otel/attribute"
)

// HandleWebhookInput holds the raw webhook payload and signature header.
type HandleWebhookInput struct {
	Payload   []byte
	Signature string
}

// HandleWebhookUseCase processes Stripe webhook events with idempotent event handling.
type HandleWebhookUseCase struct {
	gateway   providers.PaymentGateway
	subRepo   plansrepo.SubscriptionRepository
	eventRepo plansrepo.PaymentEventRepository
	logger    providers.LoggerProvider
}

// NewHandleWebhookUseCase creates a new HandleWebhookUseCase with all required dependencies.
func NewHandleWebhookUseCase(
	gateway providers.PaymentGateway,
	subRepo plansrepo.SubscriptionRepository,
	eventRepo plansrepo.PaymentEventRepository,
	logger providers.LoggerProvider,
) *HandleWebhookUseCase {
	return &HandleWebhookUseCase{
		gateway:   gateway,
		subRepo:   subRepo,
		eventRepo: eventRepo,
		logger:    logger,
	}
}

// stripeEvent is a minimal representation of a Stripe event for webhook processing.
type stripeEvent struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Data struct {
		Object json.RawMessage `json:"object"`
	} `json:"data"`
}

// Execute verifies the webhook signature, checks for duplicate events,
// and routes the event to the appropriate handler based on event type.
func (uc *HandleWebhookUseCase) Execute(ctx context.Context, input HandleWebhookInput) error {
	ctx, span := subscriptionTracer.Start(ctx, "HandleWebhookUseCase.Execute")
	defer span.End()

	log := observability.LoggerWithTrace(ctx, uc.logger).With("usecase", "HandleWebhook")

	// 1. Verify webhook signature
	verifiedPayload, err := uc.gateway.VerifyWebhookSignature(input.Payload, input.Signature)
	if err != nil {
		log.Warn("invalid webhook signature", "error", err.Error())
		observability.RecordError(span, err)
		return plansdomain.InvalidWebhookSignature()
	}

	// 2. Parse event from verified payload
	var event stripeEvent
	if err := json.Unmarshal(verifiedPayload, &event); err != nil {
		log.Error("failed to parse webhook event", "error", err.Error())
		observability.RecordError(span, err)
		return plansdomain.InvalidWebhookPayload()
	}

	span.SetAttributes(
		attribute.String("stripe.event_id", event.ID),
		attribute.String("stripe.event_type", event.Type),
	)
	log = log.With("eventId", event.ID, "eventType", event.Type)

	// 3. Idempotency check
	existing, _ := uc.eventRepo.GetByStripeEventID(ctx, event.ID)
	if existing != nil && existing.Processed {
		log.Info("event already processed, skipping")
		return nil
	}

	// 4. Store event for tracking (skip Add if we already have a record)
	var paymentEvent *plansdomain.PaymentEvent
	if existing != nil {
		paymentEvent = existing
	} else {
		paymentEvent = &plansdomain.PaymentEvent{
			StripeEventID: event.ID,
			EventType:     event.Type,
			Payload:       verifiedPayload,
		}
		if _, err := uc.eventRepo.Add(ctx, paymentEvent); err != nil {
			log.Error("failed to store payment event", "error", err.Error())
			observability.RecordError(span, err)
			return err
		}
	}

	// 5. Route by event type
	var handlerErr error
	switch event.Type {
	case "checkout.session.completed":
		handlerErr = uc.handleCheckoutCompleted(ctx, event.Data.Object, log)
	case "invoice.paid":
		handlerErr = uc.handleInvoicePaid(ctx, event.Data.Object, log)
	case "invoice.payment_failed":
		handlerErr = uc.handleInvoicePaymentFailed(ctx, event.Data.Object, log)
	case "customer.subscription.deleted":
		handlerErr = uc.handleSubscriptionDeleted(ctx, event.Data.Object, log)
	case "customer.subscription.updated":
		handlerErr = uc.handleSubscriptionUpdated(ctx, event.Data.Object, log)
	default:
		log.Info("unhandled event type, skipping")
	}

	// 6. Mark event as processed or failed
	if handlerErr != nil {
		log.Error("webhook handler failed", "error", handlerErr.Error())
		observability.RecordError(span, handlerErr)
		_ = uc.eventRepo.MarkFailed(ctx, paymentEvent.ID, handlerErr.Error())
		return handlerErr
	}

	if err := uc.eventRepo.MarkProcessed(ctx, paymentEvent.ID); err != nil {
		log.Error("failed to mark event as processed", "error", err.Error())
		observability.RecordError(span, err)
		return err
	}

	log.Info("webhook event processed successfully")
	return nil
}

// handleCheckoutCompleted processes checkout.session.completed events.
func (uc *HandleWebhookUseCase) handleCheckoutCompleted(ctx context.Context, rawObj json.RawMessage, log providers.LoggerProvider) error {
	var obj map[string]any
	if err := json.Unmarshal(rawObj, &obj); err != nil {
		return err
	}

	stripeSubID, _ := obj["subscription"].(string)
	customerID, _ := obj["customer"].(string)

	if customerID == "" {
		log.Warn("checkout.session.completed missing customer ID")
		return nil
	}

	sub, err := uc.subRepo.GetByStripeCustomerID(ctx, customerID)
	if err != nil {
		log.Error("failed to find subscription by customer ID", "customerId", customerID, "error", err.Error())
		return err
	}
	if sub == nil {
		log.Warn("no local subscription found for customer", "customerId", customerID)
		return nil
	}

	updates := map[string]any{
		"status": string(plansdomain.StatusActive),
	}
	if stripeSubID != "" {
		updates["stripe_subscription_id"] = stripeSubID
	}

	_, err = uc.subRepo.UpdateByID(ctx, sub.ID, updates)
	return err
}

// handleInvoicePaid processes invoice.paid events.
func (uc *HandleWebhookUseCase) handleInvoicePaid(ctx context.Context, rawObj json.RawMessage, log providers.LoggerProvider) error {
	var obj map[string]any
	if err := json.Unmarshal(rawObj, &obj); err != nil {
		return err
	}

	stripeSubID, _ := obj["subscription"].(string)
	if stripeSubID == "" {
		log.Info("invoice.paid without subscription ID, skipping")
		return nil
	}

	sub, err := uc.subRepo.GetByStripeSubscriptionID(ctx, stripeSubID)
	if err != nil {
		log.Error("failed to find subscription", "stripeSubscriptionId", stripeSubID, "error", err.Error())
		return err
	}
	if sub == nil {
		log.Warn("no local subscription found", "stripeSubscriptionId", stripeSubID)
		return nil
	}

	updates := map[string]any{
		"status": string(plansdomain.StatusActive),
	}

	if periodStart, ok := obj["period_start"].(float64); ok {
		t := time.Unix(int64(periodStart), 0)
		updates["current_period_start"] = t
	}
	if periodEnd, ok := obj["period_end"].(float64); ok {
		t := time.Unix(int64(periodEnd), 0)
		updates["current_period_end"] = t
	}

	_, err = uc.subRepo.UpdateByID(ctx, sub.ID, updates)
	return err
}

// handleInvoicePaymentFailed processes invoice.payment_failed events.
func (uc *HandleWebhookUseCase) handleInvoicePaymentFailed(ctx context.Context, rawObj json.RawMessage, log providers.LoggerProvider) error {
	var obj map[string]any
	if err := json.Unmarshal(rawObj, &obj); err != nil {
		return err
	}

	stripeSubID, _ := obj["subscription"].(string)
	if stripeSubID == "" {
		return nil
	}

	sub, err := uc.subRepo.GetByStripeSubscriptionID(ctx, stripeSubID)
	if err != nil {
		return err
	}
	if sub == nil {
		log.Warn("no local subscription found", "stripeSubscriptionId", stripeSubID)
		return nil
	}

	_, err = uc.subRepo.UpdateByID(ctx, sub.ID, map[string]any{
		"status": string(plansdomain.StatusPastDue),
	})
	return err
}

// handleSubscriptionDeleted processes customer.subscription.deleted events.
func (uc *HandleWebhookUseCase) handleSubscriptionDeleted(ctx context.Context, rawObj json.RawMessage, log providers.LoggerProvider) error {
	var obj map[string]any
	if err := json.Unmarshal(rawObj, &obj); err != nil {
		return err
	}

	stripeSubID, _ := obj["id"].(string)
	if stripeSubID == "" {
		return nil
	}

	sub, err := uc.subRepo.GetByStripeSubscriptionID(ctx, stripeSubID)
	if err != nil {
		return err
	}
	if sub == nil {
		log.Warn("no local subscription found", "stripeSubscriptionId", stripeSubID)
		return nil
	}

	now := time.Now()
	_, err = uc.subRepo.UpdateByID(ctx, sub.ID, map[string]any{
		"status":      string(plansdomain.StatusCanceled),
		"canceled_at": now,
	})
	return err
}

// handleSubscriptionUpdated processes customer.subscription.updated events.
func (uc *HandleWebhookUseCase) handleSubscriptionUpdated(ctx context.Context, rawObj json.RawMessage, log providers.LoggerProvider) error {
	var obj map[string]any
	if err := json.Unmarshal(rawObj, &obj); err != nil {
		return err
	}

	stripeSubID, _ := obj["id"].(string)
	if stripeSubID == "" {
		return nil
	}

	sub, err := uc.subRepo.GetByStripeSubscriptionID(ctx, stripeSubID)
	if err != nil {
		return err
	}
	if sub == nil {
		log.Warn("no local subscription found", "stripeSubscriptionId", stripeSubID)
		return nil
	}

	updates := map[string]any{}

	if status, ok := obj["status"].(string); ok {
		updates["status"] = status
	}

	if cancelAtPeriodEnd, ok := obj["cancel_at_period_end"].(bool); ok {
		updates["cancel_at_period_end"] = cancelAtPeriodEnd
	}

	if len(updates) == 0 {
		return nil
	}

	_, err = uc.subRepo.UpdateByID(ctx, sub.ID, updates)
	return err
}
