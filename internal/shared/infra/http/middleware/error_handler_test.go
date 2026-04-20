package middleware

import (
	"encoding/json"
	"errors"
	"io"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"golang_boilerplate_module/internal/shared/domain/errs"
	"golang_boilerplate_module/internal/shared/domain/providers"

	"github.com/gofiber/fiber/v2"
)

// recordingLogger counts Error calls and captures the last error message
// and fields. Implements providers.LoggerProvider.
type recordingLogger struct {
	mu         sync.Mutex
	errorCalls int
	lastMsg    string
	lastFields []any
}

func (l *recordingLogger) Info(msg string, fields ...any)  {}
func (l *recordingLogger) Warn(msg string, fields ...any)  {}
func (l *recordingLogger) Debug(msg string, fields ...any) {}
func (l *recordingLogger) Error(msg string, fields ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.errorCalls++
	l.lastMsg = msg
	l.lastFields = fields
}
func (l *recordingLogger) With(args ...any) providers.LoggerProvider { return l }
func (l *recordingLogger) Sync() error                               { return nil }

// buildApp returns a Fiber app configured with the error handler,
// registering a single route whose handler returns the provided error.
func buildApp(logger providers.LoggerProvider, handlerErr error) *fiber.App {
	app := fiber.New(fiber.Config{ErrorHandler: NewErrorHandler(logger)})
	app.Get("/test", func(c *fiber.Ctx) error { return handlerErr })
	return app
}

// doRequest executes GET /test against the app and decodes the JSON body.
func doRequest(t *testing.T, app *fiber.App) (int, errorResponse) {
	t.Helper()
	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	var decoded errorResponse
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("decode %q: %v", string(body), err)
	}
	return resp.StatusCode, decoded
}

func TestErrorHandler_DomainError(t *testing.T) {
	app := buildApp(&recordingLogger{}, errs.Errorf(errs.EBADREQUEST, "bad input"))
	status, body := doRequest(t, app)
	if status != 400 {
		t.Errorf("status = %d, want 400", status)
	}
	if body.Error != "Bad Request" {
		t.Errorf("error = %q, want %q", body.Error, "Bad Request")
	}
	if body.Message != "bad input" {
		t.Errorf("message = %q, want %q", body.Message, "bad input")
	}
}

func TestErrorHandler_NotFoundFallback(t *testing.T) {
	app := buildApp(&recordingLogger{}, fiber.NewError(fiber.StatusNotFound, "nope"))
	status, body := doRequest(t, app)
	if status != 404 {
		t.Errorf("status = %d, want 404", status)
	}
	if body.Error != "Not Found" {
		t.Errorf("error = %q, want %q", body.Error, "Not Found")
	}
	if body.Message != "Not found" {
		t.Errorf("message = %q, want %q", body.Message, "Not found")
	}
}

func TestErrorHandler_FiberOtherError(t *testing.T) {
	logger := &recordingLogger{}
	app := buildApp(logger, fiber.NewError(fiber.StatusInternalServerError, "raw fiber detail"))
	status, body := doRequest(t, app)
	if status != 500 {
		t.Errorf("status = %d, want 500", status)
	}
	if body.Message != "Internal server error" {
		t.Errorf("message = %q, want %q", body.Message, "Internal server error")
	}
	if strings.Contains(body.Message, "raw fiber detail") {
		t.Errorf("message leaked raw fiber detail: %q", body.Message)
	}
	if logger.errorCalls != 1 {
		t.Errorf("errorCalls = %d, want 1 (fiber 500 is Reportable)", logger.errorCalls)
	}
}

func TestErrorHandler_UnknownError(t *testing.T) {
	logger := &recordingLogger{}
	app := buildApp(logger, errors.New("secret connection string"))
	status, body := doRequest(t, app)
	if status != 500 {
		t.Errorf("status = %d, want 500", status)
	}
	if strings.Contains(body.Message, "secret connection string") {
		t.Errorf("T-8-01 violation: body leaked raw error: %q", body.Message)
	}
	if body.Message != "Internal server error" {
		t.Errorf("message = %q, want %q", body.Message, "Internal server error")
	}
	if logger.errorCalls != 1 {
		t.Errorf("errorCalls = %d, want 1 (unknown errors are Reportable)", logger.errorCalls)
	}
}

func TestErrorHandler_ReportableLogs(t *testing.T) {
	logger := &recordingLogger{}
	app := buildApp(logger, &errs.Error{Code: errs.EINTERNAL, Message: "boom", Reportable: true})
	_, _ = doRequest(t, app)
	if logger.errorCalls != 1 {
		t.Errorf("errorCalls = %d, want 1", logger.errorCalls)
	}
	if logger.lastMsg != "boom" {
		t.Errorf("lastMsg = %q, want %q", logger.lastMsg, "boom")
	}
	// Fields should contain "code" → "internal"
	found := false
	for i := 0; i+1 < len(logger.lastFields); i += 2 {
		if logger.lastFields[i] == "code" && logger.lastFields[i+1] == "internal" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected logger fields to contain code=internal, got %v", logger.lastFields)
	}
}

func TestErrorHandler_NonReportableDoesNotLog(t *testing.T) {
	logger := &recordingLogger{}
	app := buildApp(logger, errs.Errorf(errs.EBADREQUEST, "nope"))
	_, _ = doRequest(t, app)
	if logger.errorCalls != 0 {
		t.Errorf("errorCalls = %d, want 0 (non-reportable)", logger.errorCalls)
	}
}

func TestErrorHandler_WrappedCauseNotLeaked(t *testing.T) {
	secret := errors.New("db connection string with secret")
	wrapped := errs.Wrap(errs.EINTERNAL, secret, "operation failed")
	wrapped.Reportable = true
	app := buildApp(&recordingLogger{}, wrapped)
	_, body := doRequest(t, app)
	if body.Message != "operation failed" {
		t.Errorf("message = %q, want %q", body.Message, "operation failed")
	}
	if strings.Contains(body.Message, "secret") {
		t.Errorf("T-8-01 violation: Cause leaked into body: %q", body.Message)
	}
}

func TestErrorHandler_MetadataNotLeaked(t *testing.T) {
	e := &errs.Error{
		Code:     errs.EBADREQUEST,
		Message:  "x",
		Metadata: map[string]any{"secret": "abc"},
	}
	app := buildApp(&recordingLogger{}, e)
	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if strings.Contains(string(raw), "secret") || strings.Contains(string(raw), "abc") {
		t.Errorf("T-8-02 violation: Metadata leaked into body: %s", string(raw))
	}
}

func TestErrorHandler_StatusMapping(t *testing.T) {
	cases := []struct {
		code       errs.Code
		wantStatus int
	}{
		{errs.EUNAUTHORIZED, 401},
		{errs.EFORBIDDEN, 403},
		{errs.ENOTFOUND, 404},
		{errs.ECONFLICT, 409},
		{errs.EDUPLICATION, 409},
		{errs.EPRECONDITION, 412},
		{errs.EEXPIRED, 410},
		{errs.ERATELIMIT, 429},
		{errs.EUNAVAILABLE, 503},
	}
	for _, tc := range cases {
		app := buildApp(&recordingLogger{}, errs.Errorf(tc.code, "x"))
		status, _ := doRequest(t, app)
		if status != tc.wantStatus {
			t.Errorf("code %q → status %d, want %d", tc.code, status, tc.wantStatus)
		}
	}
}
