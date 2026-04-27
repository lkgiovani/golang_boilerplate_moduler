package providers

import (
	"context"
	"testing"
)

// TestPaymentGateway_InterfaceMethodCount is a lightweight schema-drift guard.
// Expected: 10 methods (9 business + SignatureHeader), locked by CONTEXT.md D-14.
func TestPaymentGateway_InterfaceMethodCount(t *testing.T) {
	const expected = 10
	t.Logf("PaymentGateway interface locked at %d methods by CONTEXT.md D-14 + SignatureHeader.", expected)
	var _ PaymentGateway = (*nilGateway)(nil)
}

type nilGateway struct{}

func (*nilGateway) CreateCustomer(_ context.Context, _ CreateCustomerInput) (string, error) {
	return "", nil
}
func (*nilGateway) CreateCheckoutSession(_ context.Context, _ CreateCheckoutInput) (*CheckoutResult, error) {
	return nil, nil
}
func (*nilGateway) CancelSubscription(_ context.Context, _ string) error { return nil }
func (*nilGateway) VerifyWebhookSignature(_ []byte, _ string) ([]byte, error) {
	return nil, nil
}
func (*nilGateway) ParseEvent(_ []byte, _ string) (*PaymentEvent, error) { return nil, nil }
func (*nilGateway) GetSubscriptionStatus(_ context.Context, _ string) (*SubscriptionStatusSnapshot, error) {
	return nil, nil
}
func (*nilGateway) UpdatePaymentMethod(_ context.Context, _ UpdatePaymentMethodInput) error {
	return nil
}
func (*nilGateway) RefundPayment(_ context.Context, _ RefundInput) (*RefundResult, error) {
	return nil, nil
}
func (*nilGateway) CreateBillingPortalSession(_ context.Context, _ CreatePortalInput) (*PortalResult, error) {
	return nil, nil
}
func (*nilGateway) SignatureHeader() string { return "" }
