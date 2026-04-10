package providers

import "context"

// CreateCustomerInput holds data needed to create a customer in the payment gateway.
type CreateCustomerInput struct {
	Email string
	Name  string
}

// CreateCheckoutInput holds data needed to create a checkout session.
type CreateCheckoutInput struct {
	CustomerID string
	PriceID    string
	SuccessURL string
	CancelURL  string
}

// CheckoutResult holds the result of creating a checkout session.
type CheckoutResult struct {
	SessionID  string
	SessionURL string
}

// PaymentGateway defines the contract for payment gateway adapters (e.g., Stripe).
type PaymentGateway interface {
	// CreateCustomer creates a customer in the payment gateway and returns the customer ID.
	CreateCustomer(ctx context.Context, input CreateCustomerInput) (string, error)

	// CreateCheckoutSession creates a checkout session and returns session details.
	CreateCheckoutSession(ctx context.Context, input CreateCheckoutInput) (*CheckoutResult, error)

	// CancelSubscription cancels an existing subscription by its gateway ID.
	CancelSubscription(ctx context.Context, subscriptionID string) error

	// VerifyWebhookSignature validates the payload+signature are authentic,
	// then returns the same payload bytes unchanged so the caller can unmarshal them directly.
	VerifyWebhookSignature(payload []byte, signature string) ([]byte, error)
}
