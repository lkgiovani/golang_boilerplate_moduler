package plansusecases

import (
	"context"
	"time"

	"golang_boilerplate_module/internal/modules/plans/plansdomain"
	"golang_boilerplate_module/internal/modules/plans/plansdomain/plansrepo"
	"golang_boilerplate_module/internal/shared/domain/errs"
	"golang_boilerplate_module/internal/shared/domain/providers"
	"golang_boilerplate_module/internal/shared/infra/observability"

	"go.opentelemetry.io/otel/attribute"
)

// HandleWebhookInput holds the already-parsed canonical event plus the gateway
// name (used for idempotency lookup scoping).
type HandleWebhookInput struct {
	GatewayName string
	Event       *providers.PaymentEvent
}

// HandleWebhookUseCase applies the state changes described by a canonical PaymentEvent
// with idempotency guarantees scoped by (gateway_name, gateway_event_id).
type HandleWebhookUseCase struct {
	subRepo   plansrepo.SubscriptionRepository
	eventRepo plansrepo.PaymentEventRepository
	logger    providers.LoggerProvider
}

// NewHandleWebhookUseCase wires dependencies.
func NewHandleWebhookUseCase(
	subRepo plansrepo.SubscriptionRepository,
	eventRepo plansrepo.PaymentEventRepository,
	logger providers.LoggerProvider,
) *HandleWebhookUseCase {
	return &HandleWebhookUseCase{
		subRepo:   subRepo,
		eventRepo: eventRepo,
		logger:    logger,
	}
}

// Execute runs the idempotent webhook handler.
func (uc *HandleWebhookUseCase) Execute(ctx context.Context, input HandleWebhookInput) error {
	ctx, span := subscriptionTracer.Start(ctx, "HandleWebhookUseCase.Execute")
	defer span.End()

	evt := input.Event
	log := observability.LoggerWithTrace(ctx, uc.logger).With(
		"usecase", "HandleWebhook",
		"gateway", input.GatewayName,
		"eventType", string(evt.Type),
		"eventId", evt.GatewayEventID,
	)
	span.SetAttributes(
		attribute.String("gateway.name", input.GatewayName),
		attribute.String("gateway.event_id", evt.GatewayEventID),
		attribute.String("gateway.event_type", string(evt.Type)),
	)

	// 1. Idempotency: scoped by (gateway_name, gateway_event_id)
	existing, _ := uc.eventRepo.GetByGatewayEventID(ctx, input.GatewayName, evt.GatewayEventID)
	if existing != nil && existing.Processed {
		log.Info("event already processed, skipping")
		return nil
	}

	// 2. Persist record if new
	var pe *plansdomain.PaymentEvent
	if existing != nil {
		pe = existing
	} else {
		pe = &plansdomain.PaymentEvent{
			GatewayName:    input.GatewayName,
			GatewayEventID: evt.GatewayEventID,
			EventType:      string(evt.Type),
			Payload:        evt.RawPayload,
		}
		if _, err := uc.eventRepo.Add(ctx, pe); err != nil {
			log.Error("failed to store payment event", "error", err.Error())
			observability.RecordError(span, err)
			return err
		}
	}

	// 3. Route by canonical enum
	var handlerErr error
	switch evt.Type {
	case providers.PaymentEventCheckoutCompleted:
		handlerErr = uc.handleCheckoutCompleted(ctx, input.GatewayName, evt, log)
	case providers.PaymentEventPaymentSucceeded:
		handlerErr = uc.handlePaymentSucceeded(ctx, input.GatewayName, evt, log)
	case providers.PaymentEventPaymentFailed:
		handlerErr = uc.handlePaymentFailed(ctx, input.GatewayName, evt, log)
	case providers.PaymentEventSubscriptionUpdated:
		handlerErr = uc.handleSubscriptionUpdated(ctx, input.GatewayName, evt, log)
	case providers.PaymentEventSubscriptionCanceled:
		handlerErr = uc.handleSubscriptionCanceled(ctx, input.GatewayName, evt, log)
	case providers.PaymentEventUnknown:
		// D-06: Unknown events logged + marked processed (skip silently)
		log.Info("unknown gateway event type, marking processed")
	}

	// 4. Mark processed / failed
	if handlerErr != nil {
		log.Error("webhook handler failed", "error", handlerErr.Error())
		observability.RecordError(span, handlerErr)
		_ = uc.eventRepo.MarkFailed(ctx, pe.ID, handlerErr.Error())
		return handlerErr
	}
	if err := uc.eventRepo.MarkProcessed(ctx, pe.ID); err != nil {
		log.Error("failed to mark event as processed", "error", err.Error())
		observability.RecordError(span, err)
		return err
	}

	log.Info("webhook event processed successfully")
	return nil
}

func (uc *HandleWebhookUseCase) handleCheckoutCompleted(ctx context.Context, gatewayName string, evt *providers.PaymentEvent, log providers.LoggerProvider) error {
	if evt.GatewayCustomerRef == "" {
		log.Warn("checkout completed without customer ref")
		return nil
	}
	sub, err := uc.subRepo.GetByGatewayCustomerID(ctx, gatewayName, evt.GatewayCustomerRef)
	if err != nil {
		// NOTE: repo returns SubscriptionNotFound (errs.ENOTFOUND) when absent. Downgrade to warn + nil.
		if errs.ErrorCode(err) == errs.ENOTFOUND {
			log.Warn("no local subscription for gateway customer", "customerRef", evt.GatewayCustomerRef)
			return nil
		}
		return err
	}
	if sub == nil {
		log.Warn("no local subscription for gateway customer", "customerRef", evt.GatewayCustomerRef)
		return nil
	}
	updates := map[string]any{"status": string(plansdomain.StatusActive)}
	if evt.GatewaySubscriptionRef != "" {
		updates["gateway_subscription_id"] = evt.GatewaySubscriptionRef
	}
	_, err = uc.subRepo.UpdateByID(ctx, sub.ID, updates)
	return err
}

func (uc *HandleWebhookUseCase) handlePaymentSucceeded(ctx context.Context, gatewayName string, evt *providers.PaymentEvent, log providers.LoggerProvider) error {
	if evt.GatewaySubscriptionRef == "" {
		return nil
	}
	sub, err := uc.subRepo.GetByGatewaySubscriptionID(ctx, gatewayName, evt.GatewaySubscriptionRef)
	if err != nil {
		if errs.ErrorCode(err) == errs.ENOTFOUND {
			log.Warn("no local subscription", "gatewaySubRef", evt.GatewaySubscriptionRef)
			return nil
		}
		return err
	}
	if sub == nil {
		return nil
	}
	updates := map[string]any{"status": string(plansdomain.StatusActive)}
	if evt.CurrentPeriodStart != nil {
		updates["current_period_start"] = *evt.CurrentPeriodStart
	}
	if evt.CurrentPeriodEnd != nil {
		updates["current_period_end"] = *evt.CurrentPeriodEnd
	}
	_, err = uc.subRepo.UpdateByID(ctx, sub.ID, updates)
	return err
}

func (uc *HandleWebhookUseCase) handlePaymentFailed(ctx context.Context, gatewayName string, evt *providers.PaymentEvent, log providers.LoggerProvider) error {
	if evt.GatewaySubscriptionRef == "" {
		return nil
	}
	sub, err := uc.subRepo.GetByGatewaySubscriptionID(ctx, gatewayName, evt.GatewaySubscriptionRef)
	if err != nil {
		if errs.ErrorCode(err) == errs.ENOTFOUND {
			log.Warn("no local subscription", "gatewaySubRef", evt.GatewaySubscriptionRef)
			return nil
		}
		return err
	}
	if sub == nil {
		return nil
	}
	_, err = uc.subRepo.UpdateByID(ctx, sub.ID, map[string]any{
		"status": string(plansdomain.StatusPastDue),
	})
	return err
}

func (uc *HandleWebhookUseCase) handleSubscriptionUpdated(ctx context.Context, gatewayName string, evt *providers.PaymentEvent, log providers.LoggerProvider) error {
	if evt.GatewaySubscriptionRef == "" {
		return nil
	}
	sub, err := uc.subRepo.GetByGatewaySubscriptionID(ctx, gatewayName, evt.GatewaySubscriptionRef)
	if err != nil {
		if errs.ErrorCode(err) == errs.ENOTFOUND {
			log.Warn("no local subscription", "gatewaySubRef", evt.GatewaySubscriptionRef)
			return nil
		}
		return err
	}
	if sub == nil {
		return nil
	}
	updates := map[string]any{}
	if evt.Status != "" {
		updates["status"] = evt.Status
	}
	updates["cancel_at_period_end"] = evt.CancelAtPeriodEnd
	_, err = uc.subRepo.UpdateByID(ctx, sub.ID, updates)
	return err
}

func (uc *HandleWebhookUseCase) handleSubscriptionCanceled(ctx context.Context, gatewayName string, evt *providers.PaymentEvent, log providers.LoggerProvider) error {
	if evt.GatewaySubscriptionRef == "" {
		return nil
	}
	sub, err := uc.subRepo.GetByGatewaySubscriptionID(ctx, gatewayName, evt.GatewaySubscriptionRef)
	if err != nil {
		if errs.ErrorCode(err) == errs.ENOTFOUND {
			log.Warn("no local subscription", "gatewaySubRef", evt.GatewaySubscriptionRef)
			return nil
		}
		return err
	}
	if sub == nil {
		return nil
	}
	now := time.Now()
	_, err = uc.subRepo.UpdateByID(ctx, sub.ID, map[string]any{
		"status":      string(plansdomain.StatusCanceled),
		"canceled_at": now,
	})
	return err
}
