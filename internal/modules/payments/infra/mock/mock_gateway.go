package mock

import (
	"context"
	"fmt"
	"strings"

	"golang_boilerplate_module/internal/shared/domain/providers"
)

// Gateway is a programmable in-memory PaymentGateway for unit and integration tests.
// Tests assert on the Called/LastInput fields and preload the *Return fields when they
// need to script specific gateway responses.
type Gateway struct {
	// Existing methods (ported from internal/test/integration/mock_payment_gateway_test.go)
	CreateCustomerCalled     bool
	CreateCheckoutCalled     bool
	CancelSubscriptionCalled bool
	LastCustomerInput        *providers.CreateCustomerInput
	LastCheckoutInput        *providers.CreateCheckoutInput
	LastCancelSubID          string

	// ParseEvent
	ParseEventCalled bool
	ParseEventReturn *providers.PaymentEvent
	ParseEventErr    error

	// GetSubscriptionStatus
	GetSubscriptionStatusCalled bool
	GetSubscriptionStatusReturn *providers.SubscriptionStatusSnapshot
	GetSubscriptionStatusErr    error

	// UpdatePaymentMethod
	UpdatePaymentMethodCalled    bool
	UpdatePaymentMethodLastInput *providers.UpdatePaymentMethodInput
	UpdatePaymentMethodErr       error

	// RefundPayment
	RefundPaymentCalled    bool
	RefundPaymentLastInput *providers.RefundInput
	RefundPaymentReturn    *providers.RefundResult
	RefundPaymentErr       error

	// CreateBillingPortalSession
	CreateBillingPortalCalled    bool
	CreateBillingPortalLastInput *providers.CreatePortalInput
	CreateBillingPortalReturn    *providers.PortalResult
	CreateBillingPortalErr       error

	// Signature header (overridable)
	SignatureHeaderValue string
}

// New returns a Gateway satisfying providers.PaymentGateway.
func New() *Gateway {
	return &Gateway{SignatureHeaderValue: "Mock-Signature"}
}

func (m *Gateway) CreateCustomer(_ context.Context, input providers.CreateCustomerInput) (string, error) {
	m.CreateCustomerCalled = true
	m.LastCustomerInput = &input
	return "cus_mock_" + input.Email, nil
}

func (m *Gateway) CreateCheckoutSession(_ context.Context, input providers.CreateCheckoutInput) (*providers.CheckoutResult, error) {
	m.CreateCheckoutCalled = true
	m.LastCheckoutInput = &input
	return &providers.CheckoutResult{
		SessionID:  "cs_mock_session_123",
		SessionURL: "https://checkout.stripe.com/mock/session",
	}, nil
}

func (m *Gateway) CancelSubscription(_ context.Context, subscriptionID string) error {
	m.CancelSubscriptionCalled = true
	m.LastCancelSubID = subscriptionID
	return nil
}

func (m *Gateway) VerifyWebhookSignature(payload []byte, signature string) ([]byte, error) {
	if strings.HasPrefix(signature, "whsec_valid_") {
		return payload, nil
	}
	return nil, fmt.Errorf("invalid webhook signature")
}

func (m *Gateway) ParseEvent(payload []byte, signature string) (*providers.PaymentEvent, error) {
	m.ParseEventCalled = true
	if m.ParseEventErr != nil {
		return nil, m.ParseEventErr
	}
	if m.ParseEventReturn != nil {
		return m.ParseEventReturn, nil
	}
	return nil, fmt.Errorf("mock: ParseEvent not stubbed")
}

func (m *Gateway) GetSubscriptionStatus(_ context.Context, _ string) (*providers.SubscriptionStatusSnapshot, error) {
	m.GetSubscriptionStatusCalled = true
	if m.GetSubscriptionStatusErr != nil {
		return nil, m.GetSubscriptionStatusErr
	}
	return m.GetSubscriptionStatusReturn, nil
}

func (m *Gateway) UpdatePaymentMethod(_ context.Context, input providers.UpdatePaymentMethodInput) error {
	m.UpdatePaymentMethodCalled = true
	m.UpdatePaymentMethodLastInput = &input
	return m.UpdatePaymentMethodErr
}

func (m *Gateway) RefundPayment(_ context.Context, input providers.RefundInput) (*providers.RefundResult, error) {
	m.RefundPaymentCalled = true
	m.RefundPaymentLastInput = &input
	if m.RefundPaymentErr != nil {
		return nil, m.RefundPaymentErr
	}
	if m.RefundPaymentReturn != nil {
		return m.RefundPaymentReturn, nil
	}
	return &providers.RefundResult{RefundID: "re_mock_123", Status: "succeeded"}, nil
}

func (m *Gateway) CreateBillingPortalSession(_ context.Context, input providers.CreatePortalInput) (*providers.PortalResult, error) {
	m.CreateBillingPortalCalled = true
	m.CreateBillingPortalLastInput = &input
	if m.CreateBillingPortalErr != nil {
		return nil, m.CreateBillingPortalErr
	}
	if m.CreateBillingPortalReturn != nil {
		return m.CreateBillingPortalReturn, nil
	}
	return &providers.PortalResult{URL: "https://billing.stripe.com/mock/portal"}, nil
}

func (m *Gateway) SignatureHeader() string {
	if m.SignatureHeaderValue == "" {
		return "Mock-Signature"
	}
	return m.SignatureHeaderValue
}

// Compile-time check: Gateway satisfies the interface.
var _ providers.PaymentGateway = (*Gateway)(nil)
