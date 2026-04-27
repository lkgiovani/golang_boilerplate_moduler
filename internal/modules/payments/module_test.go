package payments

import (
	"strings"
	"testing"

	"golang_boilerplate_module/internal/config"
)

func TestRegistry_ContainsStripe(t *testing.T) {
	if _, ok := registry["stripe"]; !ok {
		t.Fatal(`registry missing "stripe" entry`)
	}
}

func TestProvidePaymentGateway_StripeOK(t *testing.T) {
	cfg := &config.Config{
		PaymentGateway: config.PaymentGatewayConfig{Name: "stripe"},
		Stripe:         config.StripeConfig{SecretKey: "sk_test_x", WebhookSecret: "whsec_x"},
	}
	gw, err := providePaymentGateway(cfg)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if gw == nil {
		t.Fatal("expected non-nil gateway")
	}
}

func TestProvidePaymentGateway_UnknownFails(t *testing.T) {
	cfg := &config.Config{
		PaymentGateway: config.PaymentGatewayConfig{Name: "bogus"},
	}
	_, err := providePaymentGateway(cfg)
	if err == nil {
		t.Fatal("expected error for unknown gateway")
	}
	if !strings.Contains(err.Error(), "bogus") {
		t.Errorf("error should mention unknown name, got: %v", err)
	}
}
