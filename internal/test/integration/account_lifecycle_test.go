package integration

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"testing"
)

// queryToken fetches the latest token value from the given token table for an email.
func queryToken(t *testing.T, table, email string) string {
	t.Helper()
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	var token string
	q := fmt.Sprintf("SELECT token FROM %s WHERE email = $1 ORDER BY id DESC LIMIT 1", table)
	if err := db.QueryRow(q, email).Scan(&token); err != nil {
		t.Fatalf("query %s: %v", table, err)
	}
	return token
}

// extractTokenFromMailhogBody pulls the token= query param out of the rendered email body.
func extractTokenFromMailhogBody(t *testing.T, body string) string {
	t.Helper()
	re := regexp.MustCompile(`token=([a-f0-9]+)`)
	m := re.FindStringSubmatch(body)
	if len(m) != 2 {
		t.Fatalf("could not extract token from body: %s", body)
	}
	return m[1]
}

// emailVerifiedForUser returns the email_verified column for the given email.
func emailVerifiedForUser(t *testing.T, email string) bool {
	t.Helper()
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	var verified bool
	if err := db.QueryRow("SELECT email_verified FROM users WHERE email = $1", email).Scan(&verified); err != nil {
		t.Fatalf("query email_verified: %v", err)
	}
	return verified
}

func TestConfirmEmail_Success(t *testing.T) {
	truncateUsers(t)
	deleteAllMailhogMessages(t)

	_ = registerUser(t, "Alice", "alice-confirm@example.com", "password123")

	// registration should have dispatched a confirmation email
	msgs := fetchMailhogMessages(t)
	if msgs.Total == 0 {
		t.Fatalf("expected confirmation email, got 0 messages")
	}

	token := queryToken(t, "email_verification_tokens", "alice-confirm@example.com")
	if token == "" {
		t.Fatal("no verification token persisted")
	}

	body := fmt.Sprintf(`{"token":%q}`, token)
	req, _ := http.NewRequest(http.MethodPost, "/auth/confirm-email", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := request(req)
	if err != nil {
		t.Fatalf("confirm-email request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var e map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&e)
		t.Fatalf("expected 200, got %d: %v", resp.StatusCode, e)
	}

	if !emailVerifiedForUser(t, "alice-confirm@example.com") {
		t.Fatal("email_verified should be true after confirm")
	}
}

func TestConfirmEmail_InvalidToken(t *testing.T) {
	truncateUsers(t)

	body := `{"token":"not-a-real-token"}`
	req, _ := http.NewRequest(http.MethodPost, "/auth/confirm-email", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := request(req)
	if err != nil {
		t.Fatalf("confirm-email request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func TestConfirmEmail_AlreadyUsed(t *testing.T) {
	truncateUsers(t)
	deleteAllMailhogMessages(t)

	_ = registerUser(t, "Bob", "bob-used@example.com", "password123")
	token := queryToken(t, "email_verification_tokens", "bob-used@example.com")

	body := fmt.Sprintf(`{"token":%q}`, token)
	for i, wantStatus := range []int{http.StatusOK, http.StatusUnprocessableEntity} {
		req, _ := http.NewRequest(http.MethodPost, "/auth/confirm-email", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := request(req)
		if err != nil {
			t.Fatalf("attempt %d: %v", i, err)
		}
		if resp.StatusCode != wantStatus {
			t.Fatalf("attempt %d: expected %d, got %d", i, wantStatus, resp.StatusCode)
		}
		resp.Body.Close()
	}
}

func TestForgotAndResetPassword_Success(t *testing.T) {
	truncateUsers(t)
	deleteAllMailhogMessages(t)

	_ = registerUser(t, "Carol", "carol-reset@example.com", "originalPassword123")
	deleteAllMailhogMessages(t) // drop the confirmation email

	// forgot
	req, _ := http.NewRequest(http.MethodPost, "/auth/forgot-password",
		bytes.NewBufferString(`{"email":"carol-reset@example.com"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := request(req)
	if err != nil {
		t.Fatalf("forgot request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("forgot: expected 200, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	token := queryToken(t, "password_reset_tokens", "carol-reset@example.com")
	if token == "" {
		t.Fatal("no reset token persisted")
	}

	// reset
	body := fmt.Sprintf(`{"token":%q,"new_password":"brandNewPassword456"}`, token)
	req, _ = http.NewRequest(http.MethodPost, "/auth/reset-password", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err = request(req)
	if err != nil {
		t.Fatalf("reset request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		var e map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&e)
		t.Fatalf("reset: expected 200, got %d: %v", resp.StatusCode, e)
	}
	resp.Body.Close()

	// login with the new password must succeed
	_ = loginUser(t, "carol-reset@example.com", "brandNewPassword456")
}

func TestForgotPassword_UnknownEmail_Silent(t *testing.T) {
	truncateUsers(t)

	req, _ := http.NewRequest(http.MethodPost, "/auth/forgot-password",
		bytes.NewBufferString(`{"email":"nobody@example.com"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := request(req)
	if err != nil {
		t.Fatalf("forgot request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 (silent), got %d", resp.StatusCode)
	}
}

func TestChangePassword_Success(t *testing.T) {
	truncateUsers(t)
	deleteAllMailhogMessages(t)

	reg := registerUser(t, "Dave", "dave-change@example.com", "currentPassword1")

	body := `{"current_password":"currentPassword1","new_password":"newPassword12345"}`
	req, _ := http.NewRequest(http.MethodPut, "/auth/change-password", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+reg.TokenPair.AccessToken)
	resp, err := request(req)
	if err != nil {
		t.Fatalf("change-password request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		var e map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&e)
		t.Fatalf("expected 200, got %d: %v", resp.StatusCode, e)
	}
	resp.Body.Close()

	_ = loginUser(t, "dave-change@example.com", "newPassword12345")
}

func TestChangePassword_WrongCurrent(t *testing.T) {
	truncateUsers(t)
	deleteAllMailhogMessages(t)

	reg := registerUser(t, "Eve", "eve-change@example.com", "correctOne1")

	body := `{"current_password":"wrongOne1","new_password":"newPassword12345"}`
	req, _ := http.NewRequest(http.MethodPut, "/auth/change-password", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+reg.TokenPair.AccessToken)
	resp, err := request(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

func TestGetMe_Success(t *testing.T) {
	truncateUsers(t)
	deleteAllMailhogMessages(t)

	reg := registerUser(t, "Frank", "frank-me@example.com", "password123")

	req, _ := http.NewRequest(http.MethodGet, "/api/users/me", nil)
	req.Header.Set("Authorization", "Bearer "+reg.TokenPair.AccessToken)
	resp, err := request(req)
	if err != nil {
		t.Fatalf("me request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var out map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&out)
	if out["email"] != "frank-me@example.com" {
		t.Fatalf("expected email frank-me@example.com, got %v", out["email"])
	}
}

func TestGetMe_Unauthorized(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/api/users/me", nil)
	resp, err := request(req)
	if err != nil {
		t.Fatalf("me request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

func TestUpdateMe_Success(t *testing.T) {
	truncateUsers(t)
	deleteAllMailhogMessages(t)

	reg := registerUser(t, "Grace", "grace-me@example.com", "password123")

	body := `{"name":"Grace Hopper","img_url":"https://cdn.example.com/g.png"}`
	req, _ := http.NewRequest(http.MethodPut, "/api/users/me", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+reg.TokenPair.AccessToken)
	resp, err := request(req)
	if err != nil {
		t.Fatalf("update-me request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		var e map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&e)
		t.Fatalf("expected 200, got %d: %v", resp.StatusCode, e)
	}

	var out map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&out)
	if out["name"] != "Grace Hopper" {
		t.Fatalf("expected name Grace Hopper, got %v", out["name"])
	}
	if out["img_url"] != "https://cdn.example.com/g.png" {
		t.Fatalf("expected img_url, got %v", out["img_url"])
	}
}

// _ keeps the import used even if extractTokenFromMailhogBody is unused
var _ = extractTokenFromMailhogBody
