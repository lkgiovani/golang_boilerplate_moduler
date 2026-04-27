package integration

import (
	"bytes"
	"database/sql"
	"net/http"
	"strings"
	"testing"

	"golang_boilerplate_module/internal/shared/domain/providers"
)

func TestWebhook_MissingSignature(t *testing.T) {
	resetMockGateway(t)
	body := `{"id":"evt_test","type":"checkout.session.completed"}`
	req, _ := http.NewRequest(http.MethodPost, "/api/webhooks/stripe", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	// No Stripe-Signature header

	resp, err := request(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestWebhook_EmptyPayload(t *testing.T) {
	resetMockGateway(t)
	req, _ := http.NewRequest(http.MethodPost, "/api/webhooks/stripe", bytes.NewBufferString(""))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Stripe-Signature", "whsec_valid_test")

	resp, err := request(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestWebhook_UnknownGateway404(t *testing.T) {
	resetMockGateway(t)
	req, _ := http.NewRequest(http.MethodPost, "/api/webhooks/mercadopago", strings.NewReader("{}"))
	req.Header.Set("Stripe-Signature", "whsec_valid_xyz")

	resp, err := request(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestWebhook_ValidCheckoutCompleted(t *testing.T) {
	t.Cleanup(func() { truncatePlans(t); truncateUsers(t) })
	resetMockGateway(t)

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	var userID int64
	err = db.QueryRow(`INSERT INTO users (name, email, password, email_verified) VALUES ('Webhook User', 'webhook@test.com', 'hashed', true) RETURNING id`).Scan(&userID)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	var planID int64
	err = db.QueryRow(`INSERT INTO plans (name, slug, price_cents, currency, billing_interval, features, gateway_price_id, gateway_name) VALUES ('Pro', 'pro', 2990, 'BRL', 'monthly', '{}', 'price_test_123', 'stripe') RETURNING id`).Scan(&planID)
	if err != nil {
		t.Fatalf("create plan: %v", err)
	}

	var subID int64
	err = db.QueryRow(`INSERT INTO subscriptions (user_id, plan_id, status, gateway_customer_id, gateway_name) VALUES ($1, $2, 'incomplete', 'cus_mock_webhook@test.com', 'stripe') RETURNING id`, userID, planID).Scan(&subID)
	if err != nil {
		t.Fatalf("create subscription: %v", err)
	}

	mockGateway.ParseEventReturn = &providers.PaymentEvent{
		Type:                   providers.PaymentEventCheckoutCompleted,
		GatewayEventID:         "evt_test_checkout_1",
		GatewayCustomerRef:     "cus_mock_webhook@test.com",
		GatewaySubscriptionRef: "sub_mock_123",
	}

	req, _ := http.NewRequest(http.MethodPost, "/api/webhooks/stripe", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Stripe-Signature", "whsec_valid_test")

	resp, err := request(req)
	if err != nil {
		t.Fatalf("webhook request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var status string
	var subRef sql.NullString
	err = db.QueryRow("SELECT status, gateway_subscription_id FROM subscriptions WHERE id = $1", subID).Scan(&status, &subRef)
	if err != nil {
		t.Fatalf("query subscription: %v", err)
	}
	if status != "active" {
		t.Fatalf("expected status=active, got %s", status)
	}
	if !subRef.Valid || subRef.String != "sub_mock_123" {
		t.Fatalf("expected gateway_subscription_id=sub_mock_123, got %v", subRef)
	}
}

func TestWebhook_PaymentEventIdempotency(t *testing.T) {
	t.Cleanup(func() { truncatePlans(t); truncateUsers(t) })
	resetMockGateway(t)

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	var userID int64
	err = db.QueryRow(`INSERT INTO users (name, email, password, email_verified) VALUES ('Idempotent User', 'idempotent@test.com', 'hashed', true) RETURNING id`).Scan(&userID)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	var planID int64
	err = db.QueryRow(`INSERT INTO plans (name, slug, price_cents, currency, billing_interval, features, gateway_name) VALUES ('Basic', 'basic', 990, 'BRL', 'monthly', '{}', 'stripe') RETURNING id`).Scan(&planID)
	if err != nil {
		t.Fatalf("create plan: %v", err)
	}

	if _, err := db.Exec(`INSERT INTO subscriptions (user_id, plan_id, status, gateway_customer_id, gateway_name) VALUES ($1, $2, 'incomplete', 'cus_mock_idempotent@test.com', 'stripe')`, userID, planID); err != nil {
		t.Fatalf("create subscription: %v", err)
	}

	mockGateway.ParseEventReturn = &providers.PaymentEvent{
		Type:                   providers.PaymentEventCheckoutCompleted,
		GatewayEventID:         "evt_test_idempotent",
		GatewayCustomerRef:     "cus_mock_idempotent@test.com",
		GatewaySubscriptionRef: "sub_mock_idempotent",
	}

	req1, _ := http.NewRequest(http.MethodPost, "/api/webhooks/stripe", bytes.NewBufferString(`{}`))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("Stripe-Signature", "whsec_valid_test")
	resp1, err := request(req1)
	if err != nil {
		t.Fatalf("first webhook: %v", err)
	}
	resp1.Body.Close()
	if resp1.StatusCode != http.StatusOK {
		t.Fatalf("first webhook: expected 200, got %d", resp1.StatusCode)
	}

	req2, _ := http.NewRequest(http.MethodPost, "/api/webhooks/stripe", bytes.NewBufferString(`{}`))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Stripe-Signature", "whsec_valid_test")
	resp2, err := request(req2)
	if err != nil {
		t.Fatalf("second webhook: %v", err)
	}
	resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("second webhook: expected 200, got %d", resp2.StatusCode)
	}

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM payment_events WHERE gateway_event_id = 'evt_test_idempotent' AND gateway_name = 'stripe'").Scan(&count)
	if err != nil {
		t.Fatalf("count query: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 payment_event row, got %d", count)
	}
}

func TestWebhook_SubscriptionDeleted(t *testing.T) {
	t.Cleanup(func() { truncatePlans(t); truncateUsers(t) })
	resetMockGateway(t)

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	var userID int64
	err = db.QueryRow(`INSERT INTO users (name, email, password, email_verified) VALUES ('Delete Sub User', 'delsub@test.com', 'hashed', true) RETURNING id`).Scan(&userID)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	var planID int64
	err = db.QueryRow(`INSERT INTO plans (name, slug, price_cents, currency, billing_interval, features, gateway_name) VALUES ('Pro', 'pro-del', 2990, 'BRL', 'monthly', '{}', 'stripe') RETURNING id`).Scan(&planID)
	if err != nil {
		t.Fatalf("create plan: %v", err)
	}

	var subID int64
	err = db.QueryRow(`INSERT INTO subscriptions (user_id, plan_id, status, gateway_subscription_id, gateway_name) VALUES ($1, $2, 'active', 'sub_mock_del_123', 'stripe') RETURNING id`, userID, planID).Scan(&subID)
	if err != nil {
		t.Fatalf("create subscription: %v", err)
	}

	mockGateway.ParseEventReturn = &providers.PaymentEvent{
		Type:                   providers.PaymentEventSubscriptionCanceled,
		GatewayEventID:         "evt_test_sub_deleted",
		GatewaySubscriptionRef: "sub_mock_del_123",
	}

	req, _ := http.NewRequest(http.MethodPost, "/api/webhooks/stripe", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Stripe-Signature", "whsec_valid_test")

	resp, err := request(req)
	if err != nil {
		t.Fatalf("webhook request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var status string
	err = db.QueryRow("SELECT status FROM subscriptions WHERE id = $1", subID).Scan(&status)
	if err != nil {
		t.Fatalf("query status: %v", err)
	}
	if status != "canceled" {
		t.Fatalf("expected status=canceled, got %s", status)
	}
}

// TestWebhook_UnknownEventTypeProcessed (D-06) verifies PaymentEventUnknown is
// persisted with processed=true and does not return an error.
func TestWebhook_UnknownEventTypeProcessed(t *testing.T) {
	t.Cleanup(func() { truncatePlans(t); truncateUsers(t) })
	resetMockGateway(t)

	mockGateway.ParseEventReturn = &providers.PaymentEvent{
		Type:           providers.PaymentEventUnknown,
		GatewayEventID: "evt_test_unknown_42",
	}

	req, _ := http.NewRequest(http.MethodPost, "/api/webhooks/stripe", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Stripe-Signature", "whsec_valid_test")

	resp, err := request(req)
	if err != nil {
		t.Fatalf("webhook request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for unknown event, got %d", resp.StatusCode)
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	var processed bool
	err = db.QueryRow("SELECT processed FROM payment_events WHERE gateway_event_id = 'evt_test_unknown_42' AND gateway_name = 'stripe'").Scan(&processed)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if !processed {
		t.Error("expected unknown event to be marked processed=true (D-06)")
	}
}
