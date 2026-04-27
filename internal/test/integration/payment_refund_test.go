package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"golang_boilerplate_module/internal/shared/domain/providers"
)

// TestRefund_AdminHappyPath issues a refund as admin and asserts gateway invocation.
func TestRefund_AdminHappyPath(t *testing.T) {
	t.Cleanup(func() { truncatePlans(t); truncateUsers(t) })
	resetMockGateway(t)

	adminToken := createAdminToken(t)
	// createAdminToken seeds admin@example.com — recover its id via the response of registerUser
	// indirectly by re-querying. Simpler: create another user as the subscription owner.
	userToken, userID := createUserToken(t, "refundee@example.com")
	_ = userToken
	planID := seedPlan(t, "pro-refund", "price_pro_refund")
	subID := seedActiveSubscription(t, userID, planID, "cus_refund", "sub_refund_123")

	mockGateway.RefundPaymentReturn = &providers.RefundResult{
		RefundID: "re_expected",
		Status:   "succeeded",
	}

	body, _ := json.Marshal(map[string]any{"charge_id": "ch_test_123", "reason": "requested_by_customer"})
	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/subscriptions/%d/refund", subID), bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+adminToken)
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

	var payload map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&payload)
	if payload["refund_id"] != "re_expected" {
		t.Errorf("expected refund_id=re_expected, got %v", payload["refund_id"])
	}
	if !mockGateway.RefundPaymentCalled {
		t.Error("expected gateway.RefundPayment to be called")
	}
}

// TestRefund_NonAdminForbidden asserts a regular user gets 403.
func TestRefund_NonAdminForbidden(t *testing.T) {
	t.Cleanup(func() { truncatePlans(t); truncateUsers(t) })
	resetMockGateway(t)

	userToken, userID := createUserToken(t, "nonadmin@example.com")
	planID := seedPlan(t, "basic-refund", "price_basic_refund")
	subID := seedActiveSubscription(t, userID, planID, "cus_x", "sub_x")

	body, _ := json.Marshal(map[string]any{"charge_id": "ch_test"})
	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/subscriptions/%d/refund", subID), bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+userToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := request(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403, got %d", resp.StatusCode)
	}
}

// TestRefund_Unauthenticated asserts no token returns 401.
func TestRefund_Unauthenticated(t *testing.T) {
	resetMockGateway(t)
	body, _ := json.Marshal(map[string]any{"charge_id": "ch_test"})
	req, _ := http.NewRequest(http.MethodPost, "/api/subscriptions/1/refund", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := request(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}
}
