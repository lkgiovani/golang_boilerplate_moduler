package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

// errorResponseBody mirrors the JSON shape returned by the error handler
// middleware: `{status, error, message}`. Integration tests use this to
// assert the externally-visible HTTP contract stays stable post Phase 8.
type errorResponseBody struct {
	Status  int    `json:"status"`
	Error   string `json:"error"`
	Message string `json:"message"`
}

// decodeErrorBody reads and decodes the JSON error response body.
func decodeErrorBody(t *testing.T, resp *http.Response) errorResponseBody {
	t.Helper()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	var body errorResponseBody
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("decode %q: %v", string(raw), err)
	}
	return body
}

// TestErrorsHTTP_Auth_DuplicateEmail — EDUPLICATION (409) with "Conflict" label.
// Verifies the per-site semantic mapping from RESEARCH F-05 reaches the HTTP
// contract: duplicate email now returns 409 Conflict, not the legacy 422.
func TestErrorsHTTP_Auth_DuplicateEmail(t *testing.T) {
	t.Cleanup(func() { truncateUsers(t) })

	_ = registerUser(t, "Dup User", "errhttp-auth-dup@example.com", "password123")

	body := `{"name":"Another","email":"errhttp-auth-dup@example.com","password":"password123"}`
	req, _ := http.NewRequest(http.MethodPost, "/auth/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := request(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409, got %d", resp.StatusCode)
	}
	eb := decodeErrorBody(t, resp)
	if eb.Status != 409 {
		t.Errorf("body.status = %d, want 409", eb.Status)
	}
	if eb.Error != "Conflict" {
		t.Errorf("body.error = %q, want %q", eb.Error, "Conflict")
	}
	if !strings.Contains(eb.Message, "errhttp-auth-dup@example.com") {
		t.Errorf("body.message should mention the conflicting email, got %q", eb.Message)
	}
}

// TestErrorsHTTP_Auth_MissingCredentials — EBADREQUEST (400) with "Bad Request" label.
func TestErrorsHTTP_Auth_MissingCredentials(t *testing.T) {
	req, _ := http.NewRequest(http.MethodPost, "/auth/login",
		bytes.NewBufferString(`{"email":"","password":""}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := request(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
	eb := decodeErrorBody(t, resp)
	if eb.Status != 400 || eb.Error != "Bad Request" {
		t.Errorf("body = %+v, want status=400 error=Bad Request", eb)
	}
	if eb.Message == "" {
		t.Errorf("body.message is empty")
	}
}

// TestErrorsHTTP_Auth_InvalidCredentials — EUNAUTHORIZED (401) with "Unauthorized" label.
func TestErrorsHTTP_Auth_InvalidCredentials(t *testing.T) {
	t.Cleanup(func() { truncateUsers(t) })

	_ = registerUser(t, "Alice", "errhttp-auth-login@example.com", "password123")

	body := `{"email":"errhttp-auth-login@example.com","password":"wrongpassword"}`
	req, _ := http.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := request(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
	eb := decodeErrorBody(t, resp)
	if eb.Error != "Unauthorized" {
		t.Errorf("body.error = %q, want %q", eb.Error, "Unauthorized")
	}
	if eb.Message != "invalid credentials" {
		t.Errorf("body.message = %q, want %q", eb.Message, "invalid credentials")
	}
}

// TestErrorsHTTP_Users_NotFound — ENOTFOUND (404). Exercises the users
// GET /api/users/:id handler with a non-existent ID.
func TestErrorsHTTP_Users_NotFound(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/api/users/999999", nil)
	resp, err := request(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
	eb := decodeErrorBody(t, resp)
	if eb.Error != "Not Found" {
		t.Errorf("body.error = %q, want %q", eb.Error, "Not Found")
	}
}

// TestErrorsHTTP_Users_InvalidID — EBADREQUEST (400). Non-numeric path param.
func TestErrorsHTTP_Users_InvalidID(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/api/users/not-a-number", nil)
	resp, err := request(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
	eb := decodeErrorBody(t, resp)
	if eb.Message != "Invalid user ID" {
		t.Errorf("body.message = %q, want %q", eb.Message, "Invalid user ID")
	}
}

// TestErrorsHTTP_Plans_AuthRequired — EUNAUTHORIZED (401) on the subscribe
// endpoint when no auth token is provided.
func TestErrorsHTTP_Plans_AuthRequired(t *testing.T) {
	body := `{"plan_slug":"premium","success_url":"https://example.com/ok","cancel_url":"https://example.com/cancel"}`
	req, _ := http.NewRequest(http.MethodPost, "/api/subscriptions/checkout",
		bytes.NewBufferString(body))
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

// TestErrorsHTTP_ResponseShape_Stable — asserts that the JSON response for
// every error flows through the single {status, error, message} shape
// established in PLAN-02. Guards against accidental shape drift (V-09).
func TestErrorsHTTP_ResponseShape_Stable(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/api/users/not-a-number", nil)
	resp, err := request(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("decode: %v", err)
	}

	// Exactly 3 keys — nothing leaked from Cause or Metadata.
	if len(decoded) != 3 {
		t.Errorf("response has %d fields, want 3 — potential leak: %v", len(decoded), decoded)
	}
	for _, required := range []string{"status", "error", "message"} {
		if _, ok := decoded[required]; !ok {
			t.Errorf("response missing %q field", required)
		}
	}

	// No unexpected fields (e.g., cause, metadata, stack).
	for _, forbidden := range []string{"cause", "metadata", "stack", "trace", "details"} {
		if _, present := decoded[forbidden]; present {
			t.Errorf("T-8-01/T-8-02 violation: response contains forbidden field %q", forbidden)
		}
	}
}

// TestErrorsHTTP_FiberNotFound_Fallback — EINTERNAL fallback path through
// toDomainError. An unknown Fiber route returns 404 via the ENOTFOUND
// fallback in error_handler.go.
func TestErrorsHTTP_FiberNotFound_Fallback(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet,
		fmt.Sprintf("/api/this-route-does-not-exist-%d", 42), nil)
	resp, err := request(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
	eb := decodeErrorBody(t, resp)
	if eb.Error != "Not Found" {
		t.Errorf("body.error = %q, want %q", eb.Error, "Not Found")
	}
}
