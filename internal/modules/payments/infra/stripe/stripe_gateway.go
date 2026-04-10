package stripe

import (
	"context"

	"golang_boilerplate_module/internal/config"
	"golang_boilerplate_module/internal/shared/domain/providers"

	stripe "github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/checkout/session"
	"github.com/stripe/stripe-go/v82/customer"
	"github.com/stripe/stripe-go/v82/subscription"
	"github.com/stripe/stripe-go/v82/webhook"
)

// StripeGateway implements the PaymentGateway interface using the Stripe API.
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

// VerifyWebhookSignature validates that the payload and signature are authentic
// using the webhook secret. Returns the original payload bytes unchanged so the
// caller can unmarshal them directly.
func (g *StripeGateway) VerifyWebhookSignature(payload []byte, signature string) ([]byte, error) {
	_, err := webhook.ConstructEvent(payload, signature, g.webhookSecret)
	if err != nil {
		return nil, err
	}
	// Return original payload -- already validated by ConstructEvent
	return payload, nil
}
