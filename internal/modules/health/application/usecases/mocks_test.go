package usecases_test

import (
	"context"

	"golang_boilerplate_module/internal/shared/domain/providers"
)

// ── HealthRepository mock ─────────────────────────────────────────────────────

type mockHealthRepo struct {
	pingFn func(ctx context.Context) (bool, error)
}

func (m *mockHealthRepo) Ping(ctx context.Context) (bool, error) {
	if m.pingFn != nil {
		return m.pingFn(ctx)
	}
	return true, nil
}

// ── LoggerProvider mock ───────────────────────────────────────────────────────

type mockLogger struct{}

func (l *mockLogger) Info(msg string, fields ...any)            {}
func (l *mockLogger) Warn(msg string, fields ...any)            {}
func (l *mockLogger) Error(msg string, fields ...any)           {}
func (l *mockLogger) Debug(msg string, fields ...any)           {}
func (l *mockLogger) Sync() error                               { return nil }
func (l *mockLogger) With(args ...any) providers.LoggerProvider { return l }
