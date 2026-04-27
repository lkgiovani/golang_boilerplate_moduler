package integration

import (
	paymentmock "golang_boilerplate_module/internal/modules/payments/infra/mock"
	"golang_boilerplate_module/internal/shared/domain/providers"
)

// MockPaymentGateway is the test-file alias for paymentmock.Gateway so existing
// integration tests can keep referencing the old name. New code should prefer
// paymentmock.Gateway directly.
type MockPaymentGateway = paymentmock.Gateway

// NewMockPaymentGateway constructs a Gateway satisfying providers.PaymentGateway.
func NewMockPaymentGateway() providers.PaymentGateway {
	return paymentmock.New()
}
