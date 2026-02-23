package integration

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestHealthz(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/healthz", nil)
	resp, err := request(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	defer resp.Body.Close()

	if body["status"] != "healthy" {
		t.Fatalf("expected status=healthy, got %q", body["status"])
	}
}

func TestReadyz_DatabaseHealthy(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/readyz", nil)
	resp, err := request(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	// With a running postgres container the readiness check must return 200.
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var body struct {
		Status     string            `json:"status"`
		Components map[string]struct {
			Status string `json:"status"`
		} `json:"components"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if body.Status != "healthy" {
		t.Fatalf("expected status=healthy, got %q", body.Status)
	}
	if db, ok := body.Components["database"]; !ok || db.Status != "healthy" {
		t.Fatalf("expected database component healthy, got %+v", body.Components)
	}
}

func TestNotFound(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/route-that-does-not-exist", nil)
	resp, err := request(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}

	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if body["error"] != "Not Found" {
		t.Fatalf("expected error=Not Found, got %v", body["error"])
	}
}

func TestRequestID_IsReturnedInResponse(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/healthz", nil)
	req.Header.Set("X-Request-ID", "my-custom-id-123")

	resp, err := request(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	got := resp.Header.Get("X-Request-ID")
	if got != "my-custom-id-123" {
		t.Fatalf("expected X-Request-ID=my-custom-id-123, got %q", got)
	}
}

func TestRequestID_GeneratedWhenMissing(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/healthz", nil)
	// No X-Request-ID header set

	resp, err := request(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	got := resp.Header.Get("X-Request-ID")
	if got == "" {
		t.Fatal("expected X-Request-ID to be generated, got empty string")
	}
}
