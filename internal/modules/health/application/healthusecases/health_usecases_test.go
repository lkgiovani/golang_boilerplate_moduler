package healthusecases_test

import (
	"context"
	"errors"
	"testing"

	"golang_boilerplate_module/internal/modules/health/application/healthusecases"
	"golang_boilerplate_module/internal/modules/health/healthdomain"
	"golang_boilerplate_module/internal/shared/domain/exceptions"
)

func TestCheckHealthUseCase_AlwaysHealthy(t *testing.T) {
	uc := healthusecases.NewCheckHealthUseCase(&mockLogger{})
	out := uc.Execute(context.Background())

	if out.Status != healthdomain.HealthStatusHealthy {
		t.Fatalf("expected status=healthy, got %q", out.Status)
	}
}

func TestCheckReadinessUseCase_DatabaseHealthy(t *testing.T) {
	repo := &mockHealthRepo{
		pingFn: func(_ context.Context) (bool, error) {
			return true, nil
		},
	}

	uc := healthusecases.NewCheckReadinessUseCase(repo, &mockLogger{})
	out, err := uc.Execute(context.Background())

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if out.Status != healthdomain.HealthStatusHealthy {
		t.Fatalf("expected status=healthy, got %q", out.Status)
	}
	db, ok := out.Components["database"]
	if !ok {
		t.Fatal("expected 'database' component in response")
	}
	if db.Status != healthdomain.HealthStatusHealthy {
		t.Fatalf("expected database status=healthy, got %q", db.Status)
	}
}

func TestCheckReadinessUseCase_DatabaseUnhealthy(t *testing.T) {
	repo := &mockHealthRepo{
		pingFn: func(_ context.Context) (bool, error) {
			return false, errors.New("connection refused")
		},
	}

	uc := healthusecases.NewCheckReadinessUseCase(repo, &mockLogger{})
	_, err := uc.Execute(context.Background())

	if err == nil {
		t.Fatal("expected error for unhealthy database, got nil")
	}

	var domainErr *exceptions.DomainError
	if !errors.As(err, &domainErr) {
		t.Fatalf("expected DomainError, got %T", err)
	}
	if domainErr.Code != exceptions.CodeServiceUnavailable {
		t.Fatalf("expected SERVICE_UNAVAILABLE, got %s", domainErr.Code)
	}
}

func TestCheckReadinessUseCase_DatabasePingFalse(t *testing.T) {
	repo := &mockHealthRepo{
		pingFn: func(_ context.Context) (bool, error) {
			return false, nil
		},
	}

	uc := healthusecases.NewCheckReadinessUseCase(repo, &mockLogger{})
	_, err := uc.Execute(context.Background())

	if err == nil {
		t.Fatal("expected SERVICE_UNAVAILABLE when ping returns false, got nil")
	}

	var domainErr *exceptions.DomainError
	if !errors.As(err, &domainErr) {
		t.Fatalf("expected DomainError, got %T", err)
	}
	if domainErr.Code != exceptions.CodeServiceUnavailable {
		t.Fatalf("expected SERVICE_UNAVAILABLE, got %s", domainErr.Code)
	}
}
