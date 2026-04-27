package integration

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"golang_boilerplate_module/internal/shared/domain/providers"
)

// TestSubscriptionStatus_ReturnsSnapshot returns the gateway-reported status snapshot.
func TestSubscriptionStatus_ReturnsSnapshot(t *testing.T) {
	t.Cleanup(func() { truncatePlans(t); truncateUsers(t) })
	resetMockGateway(t)

	userToken, userID := createUserToken(t, "status-user@example.com")
	planID := seedPlan(t, "pro-status", "price_status")
	_ = seedActiveSubscription(t, userID, planID, "cus_status", "sub_status_123")

	periodStart := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	periodEnd := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	mockGateway.GetSubscriptionStatusReturn = &providers.SubscriptionStatusSnapshot{
		Status:             "active",
		CurrentPeriodStart: &periodStart,
		CurrentPeriodEnd:   &periodEnd,
		CancelAtPeriodEnd:  false,
	}

	req, _ := http.NewRequest(http.MethodGet, "/api/subscriptions/me/status", nil)
	req.Header.Set("Authorization", "Bearer "+userToken)

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
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload["status"] != "active" {
		t.Errorf("expected status=active, got %v", payload["status"])
	}
	if payload["cancel_at_period_end"] != false {
		t.Errorf("expected cancel_at_period_end=false, got %v", payload["cancel_at_period_end"])
	}
	if !mockGateway.GetSubscriptionStatusCalled {
		t.Error("expected gateway.GetSubscriptionStatus to be called")
	}
}

// TestSubscriptionStatus_NoGatewaySubscription — sub without gateway_subscription_id returns 404.
func TestSubscriptionStatus_NoGatewaySubscription(t *testing.T) {
	t.Cleanup(func() { truncatePlans(t); truncateUsers(t) })
	resetMockGateway(t)

	userToken, userID := createUserToken(t, "status-nosub@example.com")
	planID := seedPlan(t, "free-status", "price_free_status")
	_ = seedActiveSubscription(t, userID, planID, "cus_free", "")

	req, _ := http.NewRequest(http.MethodGet, "/api/subscriptions/me/status", nil)
	req.Header.Set("Authorization", "Bearer "+userToken)

	resp, err := request(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}
