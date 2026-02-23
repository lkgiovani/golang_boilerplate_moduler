package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
)

func TestCreateUser_Success(t *testing.T) {
	t.Cleanup(func() { truncateUsers(t) })

	body := `{"name":"João Silva","email":"joao@example.com"}`
	req, _ := http.NewRequest(http.MethodPost, "/api/users",
		bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := request(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}

	var user struct {
		ID    uint   `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if user.ID == 0 {
		t.Fatal("expected non-zero ID")
	}
	if user.Name != "João Silva" {
		t.Fatalf("expected name=João Silva, got %q", user.Name)
	}
	if user.Email != "joao@example.com" {
		t.Fatalf("expected email=joao@example.com, got %q", user.Email)
	}
}

func TestCreateUser_DuplicateEmail(t *testing.T) {
	t.Cleanup(func() { truncateUsers(t) })

	body := `{"name":"Maria","email":"dup@example.com"}`
	for i := 0; i < 2; i++ {
		req, _ := http.NewRequest(http.MethodPost, "/api/users",
			bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := request(req)
		if err != nil {
			t.Fatalf("request %d: %v", i, err)
		}
		resp.Body.Close()

		if i == 0 && resp.StatusCode != http.StatusCreated {
			t.Fatalf("first request: expected 201, got %d", resp.StatusCode)
		}
		if i == 1 && resp.StatusCode != http.StatusUnprocessableEntity {
			t.Fatalf("second request: expected 422, got %d", resp.StatusCode)
		}
	}
}

func TestCreateUser_MissingFields(t *testing.T) {
	req, _ := http.NewRequest(http.MethodPost, "/api/users",
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

func TestGetUser_Success(t *testing.T) {
	t.Cleanup(func() { truncateUsers(t) })

	createReq, _ := http.NewRequest(http.MethodPost, "/api/users",
		bytes.NewBufferString(`{"name":"Ana","email":"ana@example.com"}`))
	createReq.Header.Set("Content-Type", "application/json")
	createResp, err := request(createReq)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	var created struct{ ID uint `json:"id"` }
	if err := json.NewDecoder(createResp.Body).Decode(&created); err != nil {
		t.Fatalf("decode create: %v", err)
	}
	createResp.Body.Close()

	getReq, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/users/%d", created.ID), nil)
	getResp, err := request(getReq)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer getResp.Body.Close()

	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", getResp.StatusCode)
	}

	var user struct {
		ID    uint   `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	if err := json.NewDecoder(getResp.Body).Decode(&user); err != nil {
		t.Fatalf("decode get: %v", err)
	}

	if user.ID != created.ID {
		t.Fatalf("expected ID=%d, got %d", created.ID, user.ID)
	}
	if user.Email != "ana@example.com" {
		t.Fatalf("expected email=ana@example.com, got %q", user.Email)
	}
}

func TestGetUser_NotFound(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/api/users/999999", nil)
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

func TestGetUser_InvalidID(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/api/users/not-a-number", nil)
	resp, err := request(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}