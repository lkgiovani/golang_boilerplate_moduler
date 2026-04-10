package integration

import (
	"context"
	"fmt"
	"strings"

	"golang_boilerplate_module/internal/shared/domain/providers"
)

// MockPaymentGateway implements providers.PaymentGateway for integration tests.
// It simulates Stripe API calls without making real HTTP requests.
type MockPaymentGateway struct {
	CreateCustomerCalled     bool
	CreateCheckoutCalled     bool
	CancelSubscriptionCalled bool
	LastCustomerInput        *providers.CreateCustomerInput
	LastCheckoutInput        *providers.CreateCheckoutInput
	LastCancelSubID          string
}

// NewMockPaymentGateway creates a new MockPaymentGateway that satisfies the PaymentGateway interface.
func NewMockPaymentGateway() providers.PaymentGateway {
	return &MockPaymentGateway{}
}

// CreateCustomer simulates creating a customer and returns a deterministic customer ID.
func (m *MockPaymentGateway) CreateCustomer(_ context.Context, input providers.CreateCustomerInput) (string, error) {
	m.CreateCustomerCalled = true
	m.LastCustomerInput = &input
	return "cus_mock_" + input.Email, nil
}

// CreateCheckoutSession simulates creating a checkout session and returns mock session data.
func (m *MockPaymentGateway) CreateCheckoutSession(_ context.Context, input providers.CreateCheckoutInput) (*providers.CheckoutResult, error) {
	m.CreateCheckoutCalled = true
	m.LastCheckoutInput = &input
	return &providers.CheckoutResult{
		SessionID:  "cs_mock_session_123",
		SessionURL: "https://checkout.stripe.com/mock/session",
	}, nil
}

// CancelSubscription simulates canceling a subscription.
func (m *MockPaymentGateway) CancelSubscription(_ context.Context, subscriptionID string) error {
	m.CancelSubscriptionCalled = true
	m.LastCancelSubID = subscriptionID
	return nil
}

// VerifyWebhookSignature validates the signature prefix convention used in tests.
// Accepts any signature starting with "whsec_valid_" and returns the original payload.
func (m *MockPaymentGateway) VerifyWebhookSignature(payload []byte, signature string) ([]byte, error) {
	if strings.HasPrefix(signature, "whsec_valid_") {
		return payload, nil
	}
	return nil, fmt.Errorf("invalid webhook signature")
}
