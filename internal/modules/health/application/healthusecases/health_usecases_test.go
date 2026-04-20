package healthusecases_test

import (
	"context"
	"errors"
	"testing"

	"golang_boilerplate_module/internal/modules/health/application/healthusecases"
	"golang_boilerplate_module/internal/modules/health/healthdomain"
	"golang_boilerplate_module/internal/shared/domain/errs"
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
		pingRedisFn: func(_ context.Context) bool {
			return true
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
	redis, ok := out.Components["redis"]
	if !ok {
		t.Fatal("expected 'redis' component in response")
	}
	if redis.Status != healthdomain.HealthStatusHealthy {
		t.Fatalf("expected redis status=healthy, got %q", redis.Status)
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
	if got := errs.ErrorCode(err); got != errs.EUNAVAILABLE {
		t.Fatalf("expected EUNAVAILABLE, got %s", got)
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
		t.Fatal("expected EUNAVAILABLE when ping returns false, got nil")
	}
	if got := errs.ErrorCode(err); got != errs.EUNAVAILABLE {
		t.Fatalf("expected EUNAVAILABLE, got %s", got)
	}
}

func TestCheckReadinessUseCase_RedisUnhealthy(t *testing.T) {
	repo := &mockHealthRepo{
		pingFn: func(_ context.Context) (bool, error) {
			return true, nil
		},
		pingRedisFn: func(_ context.Context) bool {
			return false
		},
	}

	uc := healthusecases.NewCheckReadinessUseCase(repo, &mockLogger{})
	_, err := uc.Execute(context.Background())

	if err == nil {
		t.Fatal("expected EUNAVAILABLE when redis is unhealthy, got nil")
	}
	if got := errs.ErrorCode(err); got != errs.EUNAVAILABLE {
		t.Fatalf("expected EUNAVAILABLE, got %s", got)
	}
}
