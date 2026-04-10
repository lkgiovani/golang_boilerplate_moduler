package integration

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
)

func TestWebhook_MissingSignature(t *testing.T) {
	body := `{"id":"evt_test","type":"checkout.session.completed","data":{"object":{}}}`
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

func TestWebhook_ValidCheckoutCompleted(t *testing.T) {
	t.Cleanup(func() { truncatePlans(t); truncateUsers(t) })

	// Create plan and user with subscription via SQL
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	// Create a user
	var userID int64
	err = db.QueryRow(`INSERT INTO users (name, email, password, email_verified) VALUES ('Webhook User', 'webhook@test.com', 'hashed', true) RETURNING id`).Scan(&userID)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	// Create a plan
	var planID int64
	err = db.QueryRow(`INSERT INTO plans (name, slug, price_cents, currency, billing_interval, features, stripe_price_id) VALUES ('Pro', 'pro', 2990, 'BRL', 'monthly', '{}', 'price_test_123') RETURNING id`).Scan(&planID)
	if err != nil {
		t.Fatalf("create plan: %v", err)
	}

	// Create an incomplete subscription with stripe_customer_id
	var subID int64
	err = db.QueryRow(`INSERT INTO subscriptions (user_id, plan_id, status, stripe_customer_id) VALUES ($1, $2, 'incomplete', 'cus_mock_webhook@test.com') RETURNING id`, userID, planID).Scan(&subID)
	if err != nil {
		t.Fatalf("create subscription: %v", err)
	}

	// Send checkout.session.completed webhook
	event := map[string]any{
		"id":   "evt_test_checkout_1",
		"type": "checkout.session.completed",
		"data": map[string]any{
			"object": map[string]any{
				"subscription": "sub_mock_123",
				"customer":     "cus_mock_webhook@test.com",
			},
		},
	}
	eventBytes, _ := json.Marshal(event)

	req, _ := http.NewRequest(http.MethodPost, "/api/webhooks/stripe", bytes.NewBuffer(eventBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Stripe-Signature", "whsec_valid_test")

	resp, err := request(req)
	if err != nil {
		t.Fatalf("webhook request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errBody map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&errBody)
		t.Fatalf("expected 200, got %d, body: %v", resp.StatusCode, errBody)
	}

	// Verify subscription is now active with stripe_subscription_id
	var status string
	var stripeSubID sql.NullString
	err = db.QueryRow("SELECT status, stripe_subscription_id FROM subscriptions WHERE id = $1", subID).Scan(&status, &stripeSubID)
	if err != nil {
		t.Fatalf("query subscription: %v", err)
	}
	if status != "active" {
		t.Fatalf("expected status=active, got %s", status)
	}
	if !stripeSubID.Valid || stripeSubID.String != "sub_mock_123" {
		t.Fatalf("expected stripe_subscription_id=sub_mock_123, got %v", stripeSubID)
	}
}

func TestWebhook_PaymentEventIdempotency(t *testing.T) {
	t.Cleanup(func() { truncatePlans(t); truncateUsers(t) })

	// Create minimal data for the webhook to process
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
	err = db.QueryRow(`INSERT INTO plans (name, slug, price_cents, currency, billing_interval, features) VALUES ('Basic', 'basic', 990, 'BRL', 'monthly', '{}') RETURNING id`).Scan(&planID)
	if err != nil {
		t.Fatalf("create plan: %v", err)
	}

	err = db.QueryRow(`INSERT INTO subscriptions (user_id, plan_id, status, stripe_customer_id) VALUES ($1, $2, 'incomplete', 'cus_mock_idempotent@test.com') RETURNING id`, userID, planID).Scan(new(int64))
	if err != nil {
		t.Fatalf("create subscription: %v", err)
	}

	event := map[string]any{
		"id":   "evt_test_idempotent",
		"type": "checkout.session.completed",
		"data": map[string]any{
			"object": map[string]any{
				"subscription": "sub_mock_idempotent",
				"customer":     "cus_mock_idempotent@test.com",
			},
		},
	}
	eventBytes, _ := json.Marshal(event)

	// Send first time
	req1, _ := http.NewRequest(http.MethodPost, "/api/webhooks/stripe", bytes.NewBuffer(eventBytes))
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

	// Send second time (same event ID)
	req2, _ := http.NewRequest(http.MethodPost, "/api/webhooks/stripe", bytes.NewBuffer(eventBytes))
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

	// Verify only one payment_event row exists
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM payment_events WHERE stripe_event_id = 'evt_test_idempotent'").Scan(&count)
	if err != nil {
		t.Fatalf("count query: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 payment_event row, got %d", count)
	}
}

func TestWebhook_SubscriptionDeleted(t *testing.T) {
	t.Cleanup(func() { truncatePlans(t); truncateUsers(t) })

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
	err = db.QueryRow(`INSERT INTO plans (name, slug, price_cents, currency, billing_interval, features) VALUES ('Pro', 'pro-del', 2990, 'BRL', 'monthly', '{}') RETURNING id`).Scan(&planID)
	if err != nil {
		t.Fatalf("create plan: %v", err)
	}

	var subID int64
	err = db.QueryRow(`INSERT INTO subscriptions (user_id, plan_id, status, stripe_subscription_id) VALUES ($1, $2, 'active', 'sub_mock_del_123') RETURNING id`, userID, planID).Scan(&subID)
	if err != nil {
		t.Fatalf("create subscription: %v", err)
	}

	event := map[string]any{
		"id":   fmt.Sprintf("evt_test_sub_deleted_%d", subID),
		"type": "customer.subscription.deleted",
		"data": map[string]any{
			"object": map[string]any{
				"id": "sub_mock_del_123",
			},
		},
	}
	eventBytes, _ := json.Marshal(event)

	req, _ := http.NewRequest(http.MethodPost, "/api/webhooks/stripe", bytes.NewBuffer(eventBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Stripe-Signature", "whsec_valid_test")

	resp, err := request(req)
	if err != nil {
		t.Fatalf("webhook request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errBody map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&errBody)
		t.Fatalf("expected 200, got %d, body: %v", resp.StatusCode, errBody)
	}

	// Verify subscription status changed to canceled
	var status string
	err = db.QueryRow("SELECT status FROM subscriptions WHERE id = $1", subID).Scan(&status)
	if err != nil {
		t.Fatalf("query status: %v", err)
	}
	if status != "canceled" {
		t.Fatalf("expected status=canceled, got %s", status)
	}
}
