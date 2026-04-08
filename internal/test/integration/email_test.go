package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"golang_boilerplate_module/internal/shared/domain/providers"
)

// mailhogMessages represents the MailHog API v2 messages response.
type mailhogMessages struct {
	Total int              `json:"total"`
	Items []mailhogMessage `json:"items"`
}

type mailhogMessage struct {
	Content mailhogContent `json:"Content"`
	Raw     struct {
		From string   `json:"From"`
		To   []string `json:"To"`
	} `json:"Raw"`
}

type mailhogContent struct {
	Headers map[string][]string `json:"Headers"`
	Body    string              `json:"Body"`
}

// deleteAllMailhogMessages clears the MailHog inbox before each test.
func deleteAllMailhogMessages(t *testing.T) {
	t.Helper()
	req, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("http://%s/api/v1/messages", mailhogAPIAddr), nil)
	if err != nil {
		t.Fatalf("create delete request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("delete mailhog messages: %v", err)
	}
	defer resp.Body.Close()
}

// fetchMailhogMessages retrieves all messages from MailHog HTTP API.
func fetchMailhogMessages(t *testing.T) mailhogMessages {
	t.Helper()
	resp, err := http.Get(fmt.Sprintf("http://%s/api/v2/messages", mailhogAPIAddr))
	if err != nil {
		t.Fatalf("fetch mailhog messages: %v", err)
	}
	defer resp.Body.Close()

	var msgs mailhogMessages
	if err := json.NewDecoder(resp.Body).Decode(&msgs); err != nil {
		t.Fatalf("decode mailhog response: %v", err)
	}
	return msgs
}

func TestEmailProvider_SendConfirmation(t *testing.T) {
	deleteAllMailhogMessages(t)

	ctx := context.Background()
	err := emailProvider.Send(ctx, providers.EmailMessage{
		To:           "user@example.com",
		Subject:      "Confirm your email",
		TemplateName: "confirmation",
		Data: map[string]any{
			"Name":       "John",
			"ConfirmURL": "https://example.com/confirm?token=abc123",
		},
	})
	if err != nil {
		t.Fatalf("Send confirmation email failed: %v", err)
	}

	msgs := fetchMailhogMessages(t)
	if msgs.Total != 1 {
		t.Fatalf("expected 1 message, got %d", msgs.Total)
	}

	msg := msgs.Items[0]
	if got := msg.Content.Headers["Subject"][0]; got != "Confirm your email" {
		t.Fatalf("expected subject %q, got %q", "Confirm your email", got)
	}
	if got := msg.Content.Headers["To"][0]; got != "user@example.com" {
		t.Fatalf("expected To %q, got %q", "user@example.com", got)
	}
}

func TestEmailProvider_SendResetPassword(t *testing.T) {
	deleteAllMailhogMessages(t)

	ctx := context.Background()
	err := emailProvider.Send(ctx, providers.EmailMessage{
		To:           "reset@example.com",
		Subject:      "Reset your password",
		TemplateName: "reset_password",
		Data: map[string]any{
			"Name":     "Jane",
			"ResetURL": "https://example.com/reset?token=xyz789",
		},
	})
	if err != nil {
		t.Fatalf("Send reset password email failed: %v", err)
	}

	msgs := fetchMailhogMessages(t)
	if msgs.Total != 1 {
		t.Fatalf("expected 1 message, got %d", msgs.Total)
	}

	msg := msgs.Items[0]
	if got := msg.Content.Headers["Subject"][0]; got != "Reset your password" {
		t.Fatalf("expected subject %q, got %q", "Reset your password", got)
	}
	if got := msg.Content.Headers["To"][0]; got != "reset@example.com" {
		t.Fatalf("expected To %q, got %q", "reset@example.com", got)
	}
}

func TestEmailProvider_SendWelcome(t *testing.T) {
	deleteAllMailhogMessages(t)

	ctx := context.Background()
	err := emailProvider.Send(ctx, providers.EmailMessage{
		To:           "welcome@example.com",
		Subject:      "Welcome to SaaS Boilerplate",
		TemplateName: "welcome",
		Data: map[string]any{
			"Name": "Alice",
		},
	})
	if err != nil {
		t.Fatalf("Send welcome email failed: %v", err)
	}

	msgs := fetchMailhogMessages(t)
	if msgs.Total != 1 {
		t.Fatalf("expected 1 message, got %d", msgs.Total)
	}

	msg := msgs.Items[0]
	if got := msg.Content.Headers["Subject"][0]; got != "Welcome to SaaS Boilerplate" {
		t.Fatalf("expected subject %q, got %q", "Welcome to SaaS Boilerplate", got)
	}
	if got := msg.Content.Headers["To"][0]; got != "welcome@example.com" {
		t.Fatalf("expected To %q, got %q", "welcome@example.com", got)
	}
}

func TestEmailProvider_InvalidTemplate(t *testing.T) {
	ctx := context.Background()
	err := emailProvider.Send(ctx, providers.EmailMessage{
		To:           "test@example.com",
		Subject:      "Should fail",
		TemplateName: "nonexistent_template",
		Data:         map[string]any{},
	})
	if err == nil {
		t.Fatal("expected error for invalid template, got nil")
	}
}
