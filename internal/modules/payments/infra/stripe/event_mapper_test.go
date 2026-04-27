package stripe

import (
	"encoding/json"
	"testing"

	"golang_boilerplate_module/internal/shared/domain/providers"

	stripelib "github.com/stripe/stripe-go/v82"
)

func TestMapEvent_StripeToCanonical(t *testing.T) {
	cases := []struct {
		stripeType string
		want       providers.PaymentEventType
	}{
		{"checkout.session.completed", providers.PaymentEventCheckoutCompleted},
		{"invoice.paid", providers.PaymentEventPaymentSucceeded},
		{"invoice.payment_failed", providers.PaymentEventPaymentFailed},
		{"customer.subscription.updated", providers.PaymentEventSubscriptionUpdated},
		{"customer.subscription.deleted", providers.PaymentEventSubscriptionCanceled},
		{"some.unhandled.event", providers.PaymentEventUnknown},
	}

	for _, tc := range cases {
		t.Run(tc.stripeType, func(t *testing.T) {
			raw := []byte(`{"id":"evt_123","type":"` + tc.stripeType + `","data":{"object":{}}}`)
			evt := stripelib.Event{
				ID:   "evt_123",
				Type: stripelib.EventType(tc.stripeType),
				Data: &stripelib.EventData{Raw: json.RawMessage(`{}`)},
			}
			pe, err := mapEvent(raw, evt)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if pe.Type != tc.want {
				t.Errorf("got %q, want %q", pe.Type, tc.want)
			}
			if pe.GatewayEventID != "evt_123" {
				t.Errorf("GatewayEventID mismatch: %q", pe.GatewayEventID)
			}
			if string(pe.RawPayload) == "" {
				t.Error("RawPayload should be populated")
			}
		})
	}
}
