package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
)

// TestUpdatePaymentMethod_HappyPath updates the payment method on an active subscription.
func TestUpdatePaymentMethod_HappyPath(t *testing.T) {
	t.Cleanup(func() { truncatePlans(t); truncateUsers(t) })
	resetMockGateway(t)

	userToken, userID := createUserToken(t, "method-user@example.com")
	planID := seedPlan(t, "pro-method", "price_method")
	_ = seedActiveSubscription(t, userID, planID, "cus_method_123", "sub_method_123")

	body, _ := json.Marshal(map[string]any{"payment_method_id": "pm_card_visa"})
	req, _ := http.NewRequest(http.MethodPost, "/api/subscriptions/me/payment-method", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+userToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := request(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errBody map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&errBody)
		t.Fatalf("expected 200, got %d, body: %v", resp.StatusCode, errBody)
	}
	if !mockGateway.UpdatePaymentMethodCalled {
		t.Fatal("expected gateway.UpdatePaymentMethod to be called")
	}
	if mockGateway.UpdatePaymentMethodLastInput == nil ||
		mockGateway.UpdatePaymentMethodLastInput.PaymentMethodID != "pm_card_visa" {
		t.Errorf("unexpected payment method id: %+v", mockGateway.UpdatePaymentMethodLastInput)
	}
}

// TestUpdatePaymentMethod_InvalidBody — empty payment_method_id returns 400.
func TestUpdatePaymentMethod_InvalidBody(t *testing.T) {
	t.Cleanup(func() { truncatePlans(t); truncateUsers(t) })
	resetMockGateway(t)

	userToken, _ := createUserToken(t, "method-bad@example.com")

	body, _ := json.Marshal(map[string]any{"payment_method_id": ""})
	req, _ := http.NewRequest(http.MethodPost, "/api/subscriptions/me/payment-method", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+userToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := request(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

// TestUpdatePaymentMethod_NoActiveSubscription — user without subscription returns 404.
func TestUpdatePaymentMethod_NoActiveSubscription(t *testing.T) {
	t.Cleanup(func() { truncatePlans(t); truncateUsers(t) })
	resetMockGateway(t)

	userToken, _ := createUserToken(t, "method-nosub@example.com")

	body, _ := json.Marshal(map[string]any{"payment_method_id": "pm_card_visa"})
	req, _ := http.NewRequest(http.MethodPost, "/api/subscriptions/me/payment-method", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+userToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := request(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}
