package stripe

import (
	"encoding/json"
	"time"

	"golang_boilerplate_module/internal/shared/domain/providers"

	stripelib "github.com/stripe/stripe-go/v82"
)

// mapEvent translates a Stripe event into the canonical providers.PaymentEvent.
// Fields referenced from evt.Data.Raw are read defensively (map[string]any)
// to avoid tight coupling to stripe-go object types at the adapter boundary.
func mapEvent(raw []byte, evt stripelib.Event) (*providers.PaymentEvent, error) {
	pe := &providers.PaymentEvent{
		GatewayEventID: evt.ID,
		RawPayload:     json.RawMessage(raw),
	}

	var obj map[string]any
	if evt.Data != nil && len(evt.Data.Raw) > 0 {
		_ = json.Unmarshal(evt.Data.Raw, &obj)
	}

	switch string(evt.Type) {
	case "checkout.session.completed":
		pe.Type = providers.PaymentEventCheckoutCompleted
		pe.GatewayCustomerRef, _ = obj["customer"].(string)
		pe.GatewaySubscriptionRef, _ = obj["subscription"].(string)

	case "invoice.paid":
		pe.Type = providers.PaymentEventPaymentSucceeded
		pe.GatewaySubscriptionRef, _ = obj["subscription"].(string)
		if periodStart, ok := obj["period_start"].(float64); ok {
			t := time.Unix(int64(periodStart), 0)
			pe.CurrentPeriodStart = &t
		}
		if periodEnd, ok := obj["period_end"].(float64); ok {
			t := time.Unix(int64(periodEnd), 0)
			pe.CurrentPeriodEnd = &t
		}
		if amount, ok := obj["amount_paid"].(float64); ok {
			amt := int64(amount)
			pe.AmountCents = &amt
		}
		if cur, ok := obj["currency"].(string); ok {
			pe.Currency = cur
		}

	case "invoice.payment_failed":
		pe.Type = providers.PaymentEventPaymentFailed
		pe.GatewaySubscriptionRef, _ = obj["subscription"].(string)
		if reason, ok := obj["last_finalization_error"].(map[string]any); ok {
			if msg, ok := reason["message"].(string); ok {
				pe.FailureReason = msg
			}
		}

	case "customer.subscription.updated":
		pe.Type = providers.PaymentEventSubscriptionUpdated
		pe.GatewaySubscriptionRef, _ = obj["id"].(string)
		if status, ok := obj["status"].(string); ok {
			pe.Status = status
		}
		if cape, ok := obj["cancel_at_period_end"].(bool); ok {
			pe.CancelAtPeriodEnd = cape
		}

	case "customer.subscription.deleted":
		pe.Type = providers.PaymentEventSubscriptionCanceled
		pe.GatewaySubscriptionRef, _ = obj["id"].(string)

	default:
		pe.Type = providers.PaymentEventUnknown
	}

	return pe, nil
}
