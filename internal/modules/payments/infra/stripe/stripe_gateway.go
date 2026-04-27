package stripe

import (
	"context"
	"time"

	"golang_boilerplate_module/internal/config"
	"golang_boilerplate_module/internal/shared/domain/providers"

	stripe "github.com/stripe/stripe-go/v82"
	billingportal "github.com/stripe/stripe-go/v82/billingportal/session"
	"github.com/stripe/stripe-go/v82/checkout/session"
	"github.com/stripe/stripe-go/v82/customer"
	"github.com/stripe/stripe-go/v82/paymentmethod"
	"github.com/stripe/stripe-go/v82/refund"
	"github.com/stripe/stripe-go/v82/subscription"
	"github.com/stripe/stripe-go/v82/webhook"
)

// StripeGateway implements providers.PaymentGateway using the Stripe API.
type StripeGateway struct {
	secretKey     string
	webhookSecret string
}

// NewStripeGateway creates a new StripeGateway and sets the global Stripe API key.
func NewStripeGateway(cfg *config.Config) providers.PaymentGateway {
	stripe.Key = cfg.Stripe.SecretKey
	return &StripeGateway{
		secretKey:     cfg.Stripe.SecretKey,
		webhookSecret: cfg.Stripe.WebhookSecret,
	}
}

// CreateCustomer creates a customer in Stripe and returns the customer ID.
func (g *StripeGateway) CreateCustomer(ctx context.Context, input providers.CreateCustomerInput) (string, error) {
	params := &stripe.CustomerParams{
		Email: stripe.String(input.Email),
		Name:  stripe.String(input.Name),
	}
	c, err := customer.New(params)
	if err != nil {
		return "", err
	}
	return c.ID, nil
}

// CreateCheckoutSession creates a Stripe Checkout Session for subscription billing.
func (g *StripeGateway) CreateCheckoutSession(ctx context.Context, input providers.CreateCheckoutInput) (*providers.CheckoutResult, error) {
	params := &stripe.CheckoutSessionParams{
		Customer: stripe.String(input.CustomerID),
		Mode:     stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(input.PriceID),
				Quantity: stripe.Int64(1),
			},
		},
		SuccessURL: stripe.String(input.SuccessURL),
		CancelURL:  stripe.String(input.CancelURL),
	}
	s, err := session.New(params)
	if err != nil {
		return nil, err
	}
	return &providers.CheckoutResult{
		SessionID:  s.ID,
		SessionURL: s.URL,
	}, nil
}

// CancelSubscription cancels an existing Stripe subscription.
func (g *StripeGateway) CancelSubscription(ctx context.Context, subscriptionID string) error {
	_, err := subscription.Cancel(subscriptionID, nil)
	return err
}

// VerifyWebhookSignature validates the payload and signature and returns the payload unchanged.
func (g *StripeGateway) VerifyWebhookSignature(payload []byte, signature string) ([]byte, error) {
	_, err := webhook.ConstructEvent(payload, signature, g.webhookSecret)
	if err != nil {
		return nil, err
	}
	return payload, nil
}

// ParseEvent verifies the signature and translates the Stripe event to a canonical PaymentEvent.
func (g *StripeGateway) ParseEvent(payload []byte, signature string) (*providers.PaymentEvent, error) {
	evt, err := webhook.ConstructEvent(payload, signature, g.webhookSecret)
	if err != nil {
		return nil, err
	}
	return mapEvent(payload, evt)
}

// GetSubscriptionStatus fetches the live subscription state and projects it to the canonical shape.
func (g *StripeGateway) GetSubscriptionStatus(ctx context.Context, subscriptionID string) (*providers.SubscriptionStatusSnapshot, error) {
	sub, err := subscription.Get(subscriptionID, nil)
	if err != nil {
		return nil, err
	}
	snap := &providers.SubscriptionStatusSnapshot{
		Status:            string(sub.Status),
		CancelAtPeriodEnd: sub.CancelAtPeriodEnd,
	}
	// stripe-go v82 pins API version 2025-04-30.basil where period bounds live
	// on the first subscription item rather than the top-level subscription.
	if len(sub.Items.Data) > 0 {
		item := sub.Items.Data[0]
		if item.CurrentPeriodStart != 0 {
			t := time.Unix(item.CurrentPeriodStart, 0)
			snap.CurrentPeriodStart = &t
		}
		if item.CurrentPeriodEnd != 0 {
			t := time.Unix(item.CurrentPeriodEnd, 0)
			snap.CurrentPeriodEnd = &t
		}
	}
	return snap, nil
}

// UpdatePaymentMethod attaches a payment method to the customer and sets it as default.
func (g *StripeGateway) UpdatePaymentMethod(ctx context.Context, input providers.UpdatePaymentMethodInput) error {
	_, err := paymentmethod.Attach(input.PaymentMethodID, &stripe.PaymentMethodAttachParams{
		Customer: stripe.String(input.CustomerID),
	})
	if err != nil {
		return err
	}
	_, err = customer.Update(input.CustomerID, &stripe.CustomerParams{
		InvoiceSettings: &stripe.CustomerInvoiceSettingsParams{
			DefaultPaymentMethod: stripe.String(input.PaymentMethodID),
		},
	})
	return err
}

// RefundPayment issues a refund on the given charge.
func (g *StripeGateway) RefundPayment(ctx context.Context, input providers.RefundInput) (*providers.RefundResult, error) {
	params := &stripe.RefundParams{
		Charge: stripe.String(input.ChargeID),
	}
	if input.AmountCents != nil {
		params.Amount = stripe.Int64(*input.AmountCents)
	}
	if input.Reason != "" {
		params.Reason = stripe.String(input.Reason)
	}
	r, err := refund.New(params)
	if err != nil {
		return nil, err
	}
	return &providers.RefundResult{
		RefundID: r.ID,
		Status:   string(r.Status),
	}, nil
}

// CreateBillingPortalSession returns a URL to the Stripe-hosted customer portal.
func (g *StripeGateway) CreateBillingPortalSession(ctx context.Context, input providers.CreatePortalInput) (*providers.PortalResult, error) {
	s, err := billingportal.New(&stripe.BillingPortalSessionParams{
		Customer:  stripe.String(input.CustomerID),
		ReturnURL: stripe.String(input.ReturnURL),
	})
	if err != nil {
		return nil, err
	}
	return &providers.PortalResult{URL: s.URL}, nil
}

// SignatureHeader returns the Stripe webhook signature header name.
func (*StripeGateway) SignatureHeader() string { return "Stripe-Signature" }
