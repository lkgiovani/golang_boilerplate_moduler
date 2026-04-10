package integration

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
)

func TestPlanCRUD_AdminCreatesPlan(t *testing.T) {
	t.Cleanup(func() { truncatePlans(t); truncateUsers(t) })

	token := createAdminToken(t)

	body := `{"name":"Pro","slug":"pro","price_cents":2990,"currency":"BRL","billing_interval":"monthly","features":{"api_access":true,"export":true},"stripe_price_id":"price_test_123"}`
	req, _ := http.NewRequest(http.MethodPost, "/api/plans", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := request(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		var errBody map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&errBody)
		t.Fatalf("expected 201, got %d, body: %v", resp.StatusCode, errBody)
	}

	var plan map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&plan); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if plan["slug"] != "pro" {
		t.Fatalf("expected slug=pro, got %v", plan["slug"])
	}
	if plan["name"] != "Pro" {
		t.Fatalf("expected name=Pro, got %v", plan["name"])
	}
}

func TestPlanCRUD_ListActivePlans(t *testing.T) {
	t.Cleanup(func() { truncatePlans(t); truncateUsers(t) })

	token := createAdminToken(t)

	// Create plan 1 (active)
	body1 := `{"name":"Basic","slug":"basic","price_cents":990,"currency":"BRL","billing_interval":"monthly","features":{}}`
	req1, _ := http.NewRequest(http.MethodPost, "/api/plans", bytes.NewBufferString(body1))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("Authorization", "Bearer "+token)
	resp1, _ := request(req1)
	var plan1 map[string]any
	_ = json.NewDecoder(resp1.Body).Decode(&plan1)
	resp1.Body.Close()

	// Create plan 2 (will be deactivated)
	body2 := `{"name":"Legacy","slug":"legacy","price_cents":1990,"currency":"BRL","billing_interval":"monthly","features":{}}`
	req2, _ := http.NewRequest(http.MethodPost, "/api/plans", bytes.NewBufferString(body2))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", "Bearer "+token)
	resp2, _ := request(req2)
	var plan2 map[string]any
	_ = json.NewDecoder(resp2.Body).Decode(&plan2)
	resp2.Body.Close()

	// Deactivate plan 2
	plan2ID := fmt.Sprintf("%.0f", plan2["id"].(float64))
	delReq, _ := http.NewRequest(http.MethodDelete, "/api/plans/"+plan2ID, nil)
	delReq.Header.Set("Authorization", "Bearer "+token)
	delResp, _ := request(delReq)
	delResp.Body.Close()

	// List active plans (public, no auth needed)
	listReq, _ := http.NewRequest(http.MethodGet, "/api/plans", nil)
	listResp, err := request(listReq)
	if err != nil {
		t.Fatalf("list request: %v", err)
	}
	defer listResp.Body.Close()

	if listResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", listResp.StatusCode)
	}

	var plans []map[string]any
	if err := json.NewDecoder(listResp.Body).Decode(&plans); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if len(plans) != 1 {
		t.Fatalf("expected 1 active plan, got %d", len(plans))
	}
	if plans[0]["slug"] != "basic" {
		t.Fatalf("expected slug=basic, got %v", plans[0]["slug"])
	}
}

func TestPlanCRUD_GetPlanBySlug(t *testing.T) {
	t.Cleanup(func() { truncatePlans(t); truncateUsers(t) })

	token := createAdminToken(t)

	body := `{"name":"Pro","slug":"pro","price_cents":2990,"currency":"BRL","billing_interval":"monthly","features":{"api_access":true}}`
	createReq, _ := http.NewRequest(http.MethodPost, "/api/plans", bytes.NewBufferString(body))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Authorization", "Bearer "+token)
	createResp, _ := request(createReq)
	createResp.Body.Close()

	getReq, _ := http.NewRequest(http.MethodGet, "/api/plans/pro", nil)
	getResp, err := request(getReq)
	if err != nil {
		t.Fatalf("get request: %v", err)
	}
	defer getResp.Body.Close()

	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", getResp.StatusCode)
	}

	var plan map[string]any
	if err := json.NewDecoder(getResp.Body).Decode(&plan); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if plan["slug"] != "pro" {
		t.Fatalf("expected slug=pro, got %v", plan["slug"])
	}
}

func TestPlanCRUD_UpdatePlan(t *testing.T) {
	t.Cleanup(func() { truncatePlans(t); truncateUsers(t) })

	token := createAdminToken(t)

	body := `{"name":"Basic","slug":"basic","price_cents":990,"currency":"BRL","billing_interval":"monthly","features":{}}`
	createReq, _ := http.NewRequest(http.MethodPost, "/api/plans", bytes.NewBufferString(body))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Authorization", "Bearer "+token)
	createResp, _ := request(createReq)
	var created map[string]any
	_ = json.NewDecoder(createResp.Body).Decode(&created)
	createResp.Body.Close()

	planID := fmt.Sprintf("%.0f", created["id"].(float64))
	updateBody := `{"name":"Basic Plus"}`
	updateReq, _ := http.NewRequest(http.MethodPut, "/api/plans/"+planID, bytes.NewBufferString(updateBody))
	updateReq.Header.Set("Content-Type", "application/json")
	updateReq.Header.Set("Authorization", "Bearer "+token)

	updateResp, err := request(updateReq)
	if err != nil {
		t.Fatalf("update request: %v", err)
	}
	defer updateResp.Body.Close()

	if updateResp.StatusCode != http.StatusOK {
		var errBody map[string]any
		_ = json.NewDecoder(updateResp.Body).Decode(&errBody)
		t.Fatalf("expected 200, got %d, body: %v", updateResp.StatusCode, errBody)
	}

	var updated map[string]any
	if err := json.NewDecoder(updateResp.Body).Decode(&updated); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if updated["name"] != "Basic Plus" {
		t.Fatalf("expected name=Basic Plus, got %v", updated["name"])
	}
}

func TestPlanCRUD_DeletePlan(t *testing.T) {
	t.Cleanup(func() { truncatePlans(t); truncateUsers(t) })

	token := createAdminToken(t)

	body := `{"name":"Temp","slug":"temp","price_cents":500,"currency":"BRL","billing_interval":"monthly","features":{}}`
	createReq, _ := http.NewRequest(http.MethodPost, "/api/plans", bytes.NewBufferString(body))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Authorization", "Bearer "+token)
	createResp, _ := request(createReq)
	var created map[string]any
	_ = json.NewDecoder(createResp.Body).Decode(&created)
	createResp.Body.Close()

	planID := fmt.Sprintf("%.0f", created["id"].(float64))
	delReq, _ := http.NewRequest(http.MethodDelete, "/api/plans/"+planID, nil)
	delReq.Header.Set("Authorization", "Bearer "+token)

	delResp, err := request(delReq)
	if err != nil {
		t.Fatalf("delete request: %v", err)
	}
	defer delResp.Body.Close()

	if delResp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", delResp.StatusCode)
	}

	// Verify plan not in list
	listReq, _ := http.NewRequest(http.MethodGet, "/api/plans", nil)
	listResp, _ := request(listReq)
	defer listResp.Body.Close()

	var plans []map[string]any
	_ = json.NewDecoder(listResp.Body).Decode(&plans)
	for _, p := range plans {
		if p["slug"] == "temp" {
			t.Fatal("deleted plan should not appear in list")
		}
	}
}

func TestPlanCRUD_NonAdminForbidden(t *testing.T) {
	t.Cleanup(func() { truncatePlans(t); truncateUsers(t) })

	token, _ := createUserToken(t, "regular@example.com")

	body := `{"name":"Hack","slug":"hack","price_cents":0,"currency":"BRL","billing_interval":"monthly","features":{}}`
	req, _ := http.NewRequest(http.MethodPost, "/api/plans", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := request(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", resp.StatusCode)
	}
}

func TestPlanCRUD_UnauthenticatedForbidden(t *testing.T) {
	body := `{"name":"Hack","slug":"hack","price_cents":0,"currency":"BRL","billing_interval":"monthly","features":{}}`
	req, _ := http.NewRequest(http.MethodPost, "/api/plans", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := request(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

func TestSubscription_Checkout(t *testing.T) {
	t.Cleanup(func() { truncatePlans(t); truncateUsers(t) })

	adminToken := createAdminToken(t)

	// Create a plan with stripe_price_id
	planBody := `{"name":"Pro","slug":"pro","price_cents":2990,"currency":"BRL","billing_interval":"monthly","features":{"api_access":true},"stripe_price_id":"price_test_123"}`
	createReq, _ := http.NewRequest(http.MethodPost, "/api/plans", bytes.NewBufferString(planBody))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Authorization", "Bearer "+adminToken)
	createResp, _ := request(createReq)
	createResp.Body.Close()

	// Create a regular user
	userToken, _ := createUserToken(t, "subscriber@example.com")

	// Subscribe
	subBody := `{"plan_slug":"pro","success_url":"http://localhost/success","cancel_url":"http://localhost/cancel"}`
	subReq, _ := http.NewRequest(http.MethodPost, "/api/subscriptions/checkout", bytes.NewBufferString(subBody))
	subReq.Header.Set("Content-Type", "application/json")
	subReq.Header.Set("Authorization", "Bearer "+userToken)

	subResp, err := request(subReq)
	if err != nil {
		t.Fatalf("checkout request: %v", err)
	}
	defer subResp.Body.Close()

	if subResp.StatusCode != http.StatusOK {
		var errBody map[string]any
		_ = json.NewDecoder(subResp.Body).Decode(&errBody)
		t.Fatalf("expected 200, got %d, body: %v", subResp.StatusCode, errBody)
	}

	var result map[string]any
	if err := json.NewDecoder(subResp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if result["session_id"] != "cs_mock_session_123" {
		t.Fatalf("expected session_id=cs_mock_session_123, got %v", result["session_id"])
	}
	if result["session_url"] != "https://checkout.stripe.com/mock/session" {
		t.Fatalf("expected mock session URL, got %v", result["session_url"])
	}
}

func TestSubscription_GetMe(t *testing.T) {
	t.Cleanup(func() { truncatePlans(t); truncateUsers(t) })

	adminToken := createAdminToken(t)

	// Create a plan
	planBody := `{"name":"Pro","slug":"pro","price_cents":2990,"currency":"BRL","billing_interval":"monthly","features":{},"stripe_price_id":"price_test_123"}`
	createReq, _ := http.NewRequest(http.MethodPost, "/api/plans", bytes.NewBufferString(planBody))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Authorization", "Bearer "+adminToken)
	createResp, _ := request(createReq)
	createResp.Body.Close()

	// Create user and subscribe (creates incomplete subscription)
	userToken, userID := createUserToken(t, "getme@example.com")

	subBody := `{"plan_slug":"pro","success_url":"http://localhost/success","cancel_url":"http://localhost/cancel"}`
	subReq, _ := http.NewRequest(http.MethodPost, "/api/subscriptions/checkout", bytes.NewBufferString(subBody))
	subReq.Header.Set("Content-Type", "application/json")
	subReq.Header.Set("Authorization", "Bearer "+userToken)
	subResp, _ := request(subReq)
	subResp.Body.Close()

	// GET /api/subscriptions/me should return 404 (subscription is incomplete)
	getReq, _ := http.NewRequest(http.MethodGet, "/api/subscriptions/me", nil)
	getReq.Header.Set("Authorization", "Bearer "+userToken)
	getResp, err := request(getReq)
	if err != nil {
		t.Fatalf("get me request: %v", err)
	}
	getResp.Body.Close()

	if getResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for incomplete subscription, got %d", getResp.StatusCode)
	}

	// Manually activate the subscription in DB
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	if _, err := db.Exec("UPDATE subscriptions SET status = 'active' WHERE user_id = $1", userID); err != nil {
		t.Fatalf("update subscription: %v", err)
	}

	// GET /api/subscriptions/me should now return 200
	getReq2, _ := http.NewRequest(http.MethodGet, "/api/subscriptions/me", nil)
	getReq2.Header.Set("Authorization", "Bearer "+userToken)
	getResp2, err := request(getReq2)
	if err != nil {
		t.Fatalf("get me request 2: %v", err)
	}
	defer getResp2.Body.Close()

	if getResp2.StatusCode != http.StatusOK {
		var errBody map[string]any
		_ = json.NewDecoder(getResp2.Body).Decode(&errBody)
		t.Fatalf("expected 200, got %d, body: %v", getResp2.StatusCode, errBody)
	}

	var sub map[string]any
	if err := json.NewDecoder(getResp2.Body).Decode(&sub); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if sub["status"] != "active" {
		t.Fatalf("expected status=active, got %v", sub["status"])
	}
}

func TestSubscription_Cancel(t *testing.T) {
	t.Cleanup(func() { truncatePlans(t); truncateUsers(t) })

	adminToken := createAdminToken(t)

	// Create a plan
	planBody := `{"name":"Pro","slug":"pro","price_cents":2990,"currency":"BRL","billing_interval":"monthly","features":{},"stripe_price_id":"price_test_123"}`
	createReq, _ := http.NewRequest(http.MethodPost, "/api/plans", bytes.NewBufferString(planBody))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Authorization", "Bearer "+adminToken)
	createResp, _ := request(createReq)
	createResp.Body.Close()

	// Create user and subscribe
	userToken, userID := createUserToken(t, "cancel@example.com")

	subBody := `{"plan_slug":"pro","success_url":"http://localhost/success","cancel_url":"http://localhost/cancel"}`
	subReq, _ := http.NewRequest(http.MethodPost, "/api/subscriptions/checkout", bytes.NewBufferString(subBody))
	subReq.Header.Set("Content-Type", "application/json")
	subReq.Header.Set("Authorization", "Bearer "+userToken)
	subResp, _ := request(subReq)
	subResp.Body.Close()

	// Manually activate the subscription
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	if _, err := db.Exec("UPDATE subscriptions SET status = 'active', stripe_subscription_id = 'sub_mock_cancel' WHERE user_id = $1", userID); err != nil {
		t.Fatalf("update subscription: %v", err)
	}

	// Cancel
	cancelReq, _ := http.NewRequest(http.MethodPost, "/api/subscriptions/cancel", nil)
	cancelReq.Header.Set("Authorization", "Bearer "+userToken)

	cancelResp, err := request(cancelReq)
	if err != nil {
		t.Fatalf("cancel request: %v", err)
	}
	defer cancelResp.Body.Close()

	if cancelResp.StatusCode != http.StatusOK {
		var errBody map[string]any
		_ = json.NewDecoder(cancelResp.Body).Decode(&errBody)
		t.Fatalf("expected 200, got %d, body: %v", cancelResp.StatusCode, errBody)
	}

	// Verify status in DB
	var status string
	if err := db.QueryRow("SELECT status FROM subscriptions WHERE user_id = $1", userID).Scan(&status); err != nil {
		t.Fatalf("query status: %v", err)
	}
	if status != "canceled" {
		t.Fatalf("expected status=canceled, got %s", status)
	}
}
