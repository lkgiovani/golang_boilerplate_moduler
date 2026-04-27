package providers

import (
	"context"
	"time"
)

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

// UpdatePaymentMethodInput replaces the default payment method on a customer or subscription.
type UpdatePaymentMethodInput struct {
	CustomerID      string
	PaymentMethodID string
}

// RefundInput requests a refund on a prior charge.
type RefundInput struct {
	ChargeID    string
	AmountCents *int64
	Reason      string
}

// RefundResult holds the outcome of a refund request.
type RefundResult struct {
	RefundID string
	Status   string
}

// CreatePortalInput requests a session URL for the gateway-hosted billing portal.
type CreatePortalInput struct {
	CustomerID string
	ReturnURL  string
}

// PortalResult holds the portal session URL.
type PortalResult struct {
	URL string
}

// SubscriptionStatusSnapshot is a point-in-time projection of a subscription's state.
type SubscriptionStatusSnapshot struct {
	Status             string
	CurrentPeriodStart *time.Time
	CurrentPeriodEnd   *time.Time
	CancelAtPeriodEnd  bool
}

// PaymentGateway defines the contract every gateway adapter must satisfy.
// Exactly one implementation is active at runtime, selected by
// Config.PaymentGateway.Name at fx bootstrap (see internal/modules/payments/module.go).
type PaymentGateway interface {
	CreateCustomer(ctx context.Context, input CreateCustomerInput) (string, error)
	CreateCheckoutSession(ctx context.Context, input CreateCheckoutInput) (*CheckoutResult, error)
	CancelSubscription(ctx context.Context, subscriptionID string) error
	VerifyWebhookSignature(payload []byte, signature string) ([]byte, error)
	ParseEvent(payload []byte, signature string) (*PaymentEvent, error)
	GetSubscriptionStatus(ctx context.Context, subscriptionID string) (*SubscriptionStatusSnapshot, error)
	UpdatePaymentMethod(ctx context.Context, input UpdatePaymentMethodInput) error
	RefundPayment(ctx context.Context, input RefundInput) (*RefundResult, error)
	CreateBillingPortalSession(ctx context.Context, input CreatePortalInput) (*PortalResult, error)
	SignatureHeader() string
}
