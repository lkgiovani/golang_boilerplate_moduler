package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
)

// authTokens holds the token pair returned from register or login.
type authTokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// authRegisterResponse mirrors the register/login JSON response.
type authRegisterResponse struct {
	User struct {
		ID    int64  `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
	} `json:"user"`
	TokenPair authTokens `json:"token_pair"`
}

// authRefreshResponse mirrors the refresh JSON response.
type authRefreshResponse struct {
	TokenPair authTokens `json:"token_pair"`
}

// registerUser is a helper that registers a user and returns the full response.
func registerUser(t *testing.T, name, email, password string) authRegisterResponse {
	t.Helper()

	body := fmt.Sprintf(`{"name":%q,"email":%q,"password":%q}`, name, email, password)
	req, _ := http.NewRequest(http.MethodPost, "/auth/register",
		bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := request(req)
	if err != nil {
		t.Fatalf("register request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		var errBody map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&errBody)
		t.Fatalf("register: expected 201, got %d, body: %v", resp.StatusCode, errBody)
	}

	var result authRegisterResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("register decode: %v", err)
	}
	return result
}

// loginUser is a helper that logs in a user and returns the full response.
func loginUser(t *testing.T, email, password string) authRegisterResponse {
	t.Helper()

	body := fmt.Sprintf(`{"email":%q,"password":%q}`, email, password)
	req, _ := http.NewRequest(http.MethodPost, "/auth/login",
		bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := request(req)
	if err != nil {
		t.Fatalf("login request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errBody map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&errBody)
		t.Fatalf("login: expected 200, got %d, body: %v", resp.StatusCode, errBody)
	}

	var result authRegisterResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("login decode: %v", err)
	}
	return result
}

// refreshTokens is a helper that refreshes tokens and returns the new pair.
func refreshTokens(t *testing.T, refreshToken string) authRefreshResponse {
	t.Helper()

	body := fmt.Sprintf(`{"refresh_token":%q}`, refreshToken)
	req, _ := http.NewRequest(http.MethodPost, "/auth/refresh",
		bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := request(req)
	if err != nil {
		t.Fatalf("refresh request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errBody map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&errBody)
		t.Fatalf("refresh: expected 200, got %d, body: %v", resp.StatusCode, errBody)
	}

	var result authRefreshResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("refresh decode: %v", err)
	}
	return result
}

func TestAuth_Register_Success(t *testing.T) {
	t.Cleanup(func() { truncateUsers(t) })

	result := registerUser(t, "Auth User", "auth-register@example.com", "password123")

	if result.User.ID == 0 {
		t.Fatal("expected non-zero user ID")
	}
	if result.User.Name != "Auth User" {
		t.Fatalf("expected name=Auth User, got %q", result.User.Name)
	}
	if result.User.Email != "auth-register@example.com" {
		t.Fatalf("expected email=auth-register@example.com, got %q", result.User.Email)
	}
	if result.TokenPair.AccessToken == "" {
		t.Fatal("expected non-empty access_token")
	}
	if result.TokenPair.RefreshToken == "" {
		t.Fatal("expected non-empty refresh_token")
	}
}

func TestAuth_Register_DuplicateEmail(t *testing.T) {
	t.Cleanup(func() { truncateUsers(t) })

	_ = registerUser(t, "First User", "dup-auth@example.com", "password123")

	// Second registration with same email
	body := `{"name":"Second User","email":"dup-auth@example.com","password":"password456"}`
	req, _ := http.NewRequest(http.MethodPost, "/auth/register",
		bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := request(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", resp.StatusCode)
	}
}

func TestAuth_Register_MissingFields(t *testing.T) {
	req, _ := http.NewRequest(http.MethodPost, "/auth/register",
		bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")

	resp, err := request(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestAuth_Register_ShortPassword(t *testing.T) {
	body := `{"name":"Short Pass","email":"short@example.com","password":"1234567"}`
	req, _ := http.NewRequest(http.MethodPost, "/auth/register",
		bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := request(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestAuth_Login_Success(t *testing.T) {
	t.Cleanup(func() { truncateUsers(t) })

	_ = registerUser(t, "Login User", "login@example.com", "password123")
	result := loginUser(t, "login@example.com", "password123")

	if result.User.ID == 0 {
		t.Fatal("expected non-zero user ID")
	}
	if result.User.Email != "login@example.com" {
		t.Fatalf("expected email=login@example.com, got %q", result.User.Email)
	}
	if result.TokenPair.AccessToken == "" {
		t.Fatal("expected non-empty access_token")
	}
	if result.TokenPair.RefreshToken == "" {
		t.Fatal("expected non-empty refresh_token")
	}
}

func TestAuth_Login_WrongPassword(t *testing.T) {
	t.Cleanup(func() { truncateUsers(t) })

	_ = registerUser(t, "Wrong Pass", "wrong-pass@example.com", "password123")

	body := `{"email":"wrong-pass@example.com","password":"wrongpassword"}`
	req, _ := http.NewRequest(http.MethodPost, "/auth/login",
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

	var errBody map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&errBody)
	if errBody["message"] != "invalid credentials" {
		t.Fatalf("expected message=invalid credentials, got %v", errBody["message"])
	}
}

func TestAuth_Login_NonExistentUser(t *testing.T) {
	body := `{"email":"nonexistent@example.com","password":"password123"}`
	req, _ := http.NewRequest(http.MethodPost, "/auth/login",
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

	var errBody map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&errBody)
	if errBody["message"] != "invalid credentials" {
		t.Fatalf("expected message=invalid credentials, got %v", errBody["message"])
	}
}

func TestAuth_Refresh_Success(t *testing.T) {
	t.Cleanup(func() { truncateUsers(t) })

	reg := registerUser(t, "Refresh User", "refresh@example.com", "password123")
	login := loginUser(t, "refresh@example.com", "password123")

	result := refreshTokens(t, login.TokenPair.RefreshToken)

	if result.TokenPair.AccessToken == "" {
		t.Fatal("expected non-empty new access_token")
	}
	if result.TokenPair.RefreshToken == "" {
		t.Fatal("expected non-empty new refresh_token")
	}
	if result.TokenPair.AccessToken == reg.TokenPair.AccessToken {
		t.Fatal("expected new access_token to differ from original")
	}
	if result.TokenPair.RefreshToken == login.TokenPair.RefreshToken {
		t.Fatal("expected new refresh_token to differ from login token")
	}
}

func TestAuth_Refresh_OldTokenInvalid(t *testing.T) {
	t.Cleanup(func() { truncateUsers(t) })

	_ = registerUser(t, "Old Token User", "old-token@example.com", "password123")
	login := loginUser(t, "old-token@example.com", "password123")

	oldRefreshToken := login.TokenPair.RefreshToken

	// Refresh once to rotate the token
	_ = refreshTokens(t, oldRefreshToken)

	// Try to reuse the old refresh token - should fail because it was marked as used
	body := fmt.Sprintf(`{"refresh_token":%q}`, oldRefreshToken)
	req, _ := http.NewRequest(http.MethodPost, "/auth/refresh",
		bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := request(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 for reused token, got %d", resp.StatusCode)
	}
}

func TestAuth_Refresh_TheftDetection(t *testing.T) {
	t.Cleanup(func() { truncateUsers(t) })

	// 1. Register and login
	_ = registerUser(t, "Theft User", "theft@example.com", "password123")
	login := loginUser(t, "theft@example.com", "password123")
	oldRefreshToken := login.TokenPair.RefreshToken

	// 2. Refresh once (attacker gets new tokens, old token marked as used)
	newTokens := refreshTokens(t, oldRefreshToken)

	// 3. Try to use OLD token again (simulates theft: attacker uses stolen token)
	// This should trigger theft detection and revoke the entire family
	body := fmt.Sprintf(`{"refresh_token":%q}`, oldRefreshToken)
	req, _ := http.NewRequest(http.MethodPost, "/auth/refresh",
		bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := request(req)
	if err != nil {
		t.Fatalf("theft detection request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("theft detection: expected 401, got %d", resp.StatusCode)
	}

	// 4. Verify the new (legitimate) tokens are also revoked (family revocation)
	body = fmt.Sprintf(`{"refresh_token":%q}`, newTokens.TokenPair.RefreshToken)
	req, _ = http.NewRequest(http.MethodPost, "/auth/refresh",
		bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp2, err := request(req)
	if err != nil {
		t.Fatalf("post-theft refresh request: %v", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusUnauthorized {
		t.Fatalf("post-theft: expected 401 (family revoked), got %d", resp2.StatusCode)
	}
}

func TestAuth_Logout_Success(t *testing.T) {
	t.Cleanup(func() { truncateUsers(t) })

	_ = registerUser(t, "Logout User", "logout@example.com", "password123")
	login := loginUser(t, "logout@example.com", "password123")

	req, _ := http.NewRequest(http.MethodPost, "/auth/logout", nil)
	req.Header.Set("Authorization", "Bearer "+login.TokenPair.AccessToken)

	resp, err := request(req)
	if err != nil {
		t.Fatalf("logout request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errBody map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&errBody)
		t.Fatalf("logout: expected 200, got %d, body: %v", resp.StatusCode, errBody)
	}

	var body map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&body)
	if body["message"] != "logged out successfully" {
		t.Fatalf("expected message=logged out successfully, got %v", body["message"])
	}
}

func TestAuth_Logout_TokenBlacklisted(t *testing.T) {
	t.Cleanup(func() { truncateUsers(t) })

	_ = registerUser(t, "Blacklist User", "blacklist@example.com", "password123")
	login := loginUser(t, "blacklist@example.com", "password123")

	// Logout to blacklist the access token
	logoutReq, _ := http.NewRequest(http.MethodPost, "/auth/logout", nil)
	logoutReq.Header.Set("Authorization", "Bearer "+login.TokenPair.AccessToken)

	logoutResp, err := request(logoutReq)
	if err != nil {
		t.Fatalf("logout request: %v", err)
	}
	logoutResp.Body.Close()

	if logoutResp.StatusCode != http.StatusOK {
		t.Fatalf("logout: expected 200, got %d", logoutResp.StatusCode)
	}

	// Try using the blacklisted token on a protected endpoint
	protectedReq, _ := http.NewRequest(http.MethodPost, "/auth/logout", nil)
	protectedReq.Header.Set("Authorization", "Bearer "+login.TokenPair.AccessToken)

	protectedResp, err := request(protectedReq)
	if err != nil {
		t.Fatalf("protected request: %v", err)
	}
	defer protectedResp.Body.Close()

	if protectedResp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 for blacklisted token, got %d", protectedResp.StatusCode)
	}
}

func TestAuth_CookieMode(t *testing.T) {
	t.Cleanup(func() { truncateUsers(t) })

	_ = registerUser(t, "Cookie User", "cookie@example.com", "password123")

	body := `{"email":"cookie@example.com","password":"password123"}`
	req, _ := http.NewRequest(http.MethodPost, "/auth/login",
		bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Mode", "cookie")

	resp, err := request(req)
	if err != nil {
		t.Fatalf("cookie mode request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errBody map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&errBody)
		t.Fatalf("cookie mode: expected 200, got %d, body: %v", resp.StatusCode, errBody)
	}

	// Check that Set-Cookie headers are present
	cookies := resp.Header.Values("Set-Cookie")
	if len(cookies) == 0 {
		t.Fatal("expected Set-Cookie headers in cookie mode")
	}

	foundAccess := false
	foundRefresh := false
	for _, cookie := range cookies {
		if contains(cookie, "access_token=") {
			foundAccess = true
			if !contains(cookie, "HttpOnly") {
				t.Fatal("access_token cookie should be HttpOnly")
			}
		}
		if contains(cookie, "refresh_token=") {
			foundRefresh = true
			if !contains(cookie, "HttpOnly") {
				t.Fatal("refresh_token cookie should be HttpOnly")
			}
		}
	}

	if !foundAccess {
		t.Fatal("expected access_token cookie in response")
	}
	if !foundRefresh {
		t.Fatal("expected refresh_token cookie in response")
	}

	// Verify the JSON response does NOT contain tokens (cookie mode strips them)
	var jsonBody map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&jsonBody)
	if _, hasTokenPair := jsonBody["token_pair"]; hasTokenPair {
		t.Fatal("cookie mode response should not contain token_pair in JSON body")
	}
}

func TestAuth_Middleware_NoToken(t *testing.T) {
	// Access a protected endpoint without any token
	req, _ := http.NewRequest(http.MethodPost, "/auth/logout", nil)

	resp, err := request(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 for missing token, got %d", resp.StatusCode)
	}
}

// contains checks if a string contains a substring.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
