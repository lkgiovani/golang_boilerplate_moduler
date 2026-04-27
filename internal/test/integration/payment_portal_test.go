package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"golang_boilerplate_module/internal/shared/domain/providers"
)

// TestPortal_ReturnsURL (D-20) verifies that POST /api/subscriptions/me/portal
// returns the gateway-hosted portal URL for an authenticated user with an
// active subscription.
func TestPortal_ReturnsURL(t *testing.T) {
	t.Cleanup(func() { truncatePlans(t); truncateUsers(t) })
	resetMockGateway(t)

	userToken, userID := createUserToken(t, "portal@example.com")
	planID := seedPlan(t, "basic", "price_basic_123")
	_ = seedActiveSubscription(t, userID, planID, "cus_mock_portal", "sub_mock_portal")

	mockGateway.CreateBillingPortalReturn = &providers.PortalResult{
		URL: "https://billing.stripe.com/portal/expected",
	}

	body, _ := json.Marshal(map[string]any{"return_url": "https://app.example.com/account"})
	req, _ := http.NewRequest(http.MethodPost, "/api/subscriptions/me/portal", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+userToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := request(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var payload struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload.URL != "https://billing.stripe.com/portal/expected" {
		t.Errorf("unexpected URL: %q", payload.URL)
	}
	if !mockGateway.CreateBillingPortalCalled {
		t.Error("expected gateway.CreateBillingPortalSession to be called")
	}
}

// TestPortal_RequiresAuth ensures an unauthenticated request returns 401.
func TestPortal_RequiresAuth(t *testing.T) {
	resetMockGateway(t)
	body, _ := json.Marshal(map[string]any{"return_url": "https://app.example.com/account"})
	req, _ := http.NewRequest(http.MethodPost, "/api/subscriptions/me/portal", bytes.NewReader(body))
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
