package providers

import (
	"encoding/json"
	"time"
)

// PaymentEventType is the canonical gateway-agnostic event category.
// Adapters map provider-specific event strings (e.g., Stripe "invoice.paid")
// to one of these constants inside their ParseEvent implementation.
type PaymentEventType string

const (
	PaymentEventSubscriptionActivated PaymentEventType = "subscription_activated"
	PaymentEventSubscriptionUpdated   PaymentEventType = "subscription_updated"
	PaymentEventSubscriptionCanceled  PaymentEventType = "subscription_canceled"
	PaymentEventPaymentSucceeded      PaymentEventType = "payment_succeeded"
	PaymentEventPaymentFailed         PaymentEventType = "payment_failed"
	PaymentEventCheckoutCompleted     PaymentEventType = "checkout_completed"
	PaymentEventUnknown               PaymentEventType = "unknown"
)

// PaymentEvent is the gateway-agnostic representation of a webhook event.
// Adapter.ParseEvent returns this; HandleWebhookUseCase consumes it without
// knowing which gateway produced it.
type PaymentEvent struct {
	Type                   PaymentEventType
	GatewayEventID         string
	GatewayCustomerRef     string
	GatewaySubscriptionRef string
	Status                 string
	CurrentPeriodStart     *time.Time
	CurrentPeriodEnd       *time.Time
	CancelAtPeriodEnd      bool
	AmountCents            *int64
	Currency               string
	FailureReason          string
	RawPayload             json.RawMessage
}
