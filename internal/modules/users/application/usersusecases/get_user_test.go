package usersusecases_test

import (
	"context"
	"errors"
	"testing"

	"golang_boilerplate_module/internal/modules/users/application/usersusecases"
	"golang_boilerplate_module/internal/modules/users/usersdomain"
	"golang_boilerplate_module/internal/shared/domain/errs"
)

func TestGetUserUseCase_Success(t *testing.T) {
	expected := &usersdomain.User{ID: 42, Name: "Ana", Email: "ana@example.com"}

	repo := &mockUserRepo{
		getByIDFn: func(_ context.Context, id int64) (*usersdomain.User, error) {
			if id == 42 {
				return expected, nil
			}
			return nil, usersdomain.UserNotFound()
		},
	}

	uc := usersusecases.NewGetUserUseCase(repo, &mockLogger{})
	out, err := uc.Execute(context.Background(), 42)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if out.ID != 42 {
		t.Fatalf("expected ID=42, got %d", out.ID)
	}
	if out.Name != "Ana" {
		t.Fatalf("expected name=Ana, got %q", out.Name)
	}
	if out.Email != "ana@example.com" {
		t.Fatalf("expected email=ana@example.com, got %q", out.Email)
	}
}

func TestGetUserUseCase_NotFound(t *testing.T) {
	notFoundErr := usersdomain.UserNotFound()

	repo := &mockUserRepo{
		getByIDFn: func(_ context.Context, _ int64) (*usersdomain.User, error) {
			return nil, notFoundErr
		},
	}

	uc := usersusecases.NewGetUserUseCase(repo, &mockLogger{})
	_, err := uc.Execute(context.Background(), 999)

	if err == nil {
		t.Fatal("expected not-found error, got nil")
	}
	if got := errs.ErrorCode(err); got != errs.ENOTFOUND {
		t.Fatalf("expected ENOTFOUND, got %s", got)
	}
}

func TestGetUserUseCase_RepositoryError(t *testing.T) {
	repoErr := errors.New("timeout")

	repo := &mockUserRepo{
		getByIDFn: func(_ context.Context, _ int64) (*usersdomain.User, error) {
			return nil, repoErr
		},
	}

	uc := usersusecases.NewGetUserUseCase(repo, &mockLogger{})
	_, err := uc.Execute(context.Background(), 1)

	if err == nil {
		t.Fatal("expected repository error, got nil")
	}
	if !errors.Is(err, repoErr) {
		t.Fatalf("expected repoErr, got %v", err)
	}
}
