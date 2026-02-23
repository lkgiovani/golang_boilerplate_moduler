package usecases_test

import (
	"context"

	"golang_boilerplate_module/internal/modules/users/domain"
	"golang_boilerplate_module/internal/shared/domain/providers"
)

// ── UserRepository mock ───────────────────────────────────────────────────────

type mockUserRepo struct {
	addFn        func(ctx context.Context, u *domain.User) (*domain.User, error)
	getByIDFn    func(ctx context.Context, id uint) (*domain.User, error)
	getByEmailFn func(ctx context.Context, email string) (*domain.User, error)
	updateFn     func(ctx context.Context, id uint, updates map[string]any) (*domain.User, error)
	deleteFn     func(ctx context.Context, id uint) error
	deleteAllFn  func(ctx context.Context) error
}

func (m *mockUserRepo) Add(ctx context.Context, u *domain.User) (*domain.User, error) {
	if m.addFn != nil {
		return m.addFn(ctx, u)
	}
	return u, nil
}

func (m *mockUserRepo) GetByID(ctx context.Context, id uint) (*domain.User, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, nil
}

func (m *mockUserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	if m.getByEmailFn != nil {
		return m.getByEmailFn(ctx, email)
	}
	return nil, nil
}

func (m *mockUserRepo) UpdateByID(ctx context.Context, id uint, updates map[string]any) (*domain.User, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, id, updates)
	}
	return nil, nil
}

func (m *mockUserRepo) DeleteByID(ctx context.Context, id uint) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}

func (m *mockUserRepo) DeleteAll(ctx context.Context) error {
	if m.deleteAllFn != nil {
		return m.deleteAllFn(ctx)
	}
	return nil
}

// ── LoggerProvider mock ───────────────────────────────────────────────────────

type mockLogger struct{}

func (l *mockLogger) Info(msg string, fields ...any)            {}
func (l *mockLogger) Warn(msg string, fields ...any)            {}
func (l *mockLogger) Error(msg string, fields ...any)           {}
func (l *mockLogger) Debug(msg string, fields ...any)           {}
func (l *mockLogger) Sync() error                               { return nil }
func (l *mockLogger) With(args ...any) providers.LoggerProvider { return l }
