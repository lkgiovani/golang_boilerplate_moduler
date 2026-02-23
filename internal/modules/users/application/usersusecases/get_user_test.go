package usersusecases_test

import (
	"context"
	"errors"
	"testing"

	"golang_boilerplate_module/internal/modules/users/application/usersusecases"
	"golang_boilerplate_module/internal/modules/users/usersdomain"
	"golang_boilerplate_module/internal/shared/domain/exceptions"
)

func TestGetUserUseCase_Success(t *testing.T) {
	expected := &usersdomain.User{ID: 42, Name: "Ana", Email: "ana@example.com"}

	repo := &mockUserRepo{
		getByIDFn: func(_ context.Context, id uint) (*usersdomain.User, error) {
			if id == 42 {
				return expected, nil
			}
			return nil, exceptions.NewNotFoundException("user not found", nil)
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
	notFoundErr := exceptions.NewNotFoundException("user not found", nil)

	repo := &mockUserRepo{
		getByIDFn: func(_ context.Context, _ uint) (*usersdomain.User, error) {
			return nil, notFoundErr
		},
	}

	uc := usersusecases.NewGetUserUseCase(repo, &mockLogger{})
	_, err := uc.Execute(context.Background(), 999)

	if err == nil {
		t.Fatal("expected not-found error, got nil")
	}

	var domainErr *exceptions.DomainError
	if !errors.As(err, &domainErr) {
		t.Fatalf("expected DomainError, got %T", err)
	}
	if domainErr.Code != exceptions.CodeNotFound {
		t.Fatalf("expected NOT_FOUND, got %s", domainErr.Code)
	}
}

func TestGetUserUseCase_RepositoryError(t *testing.T) {
	repoErr := errors.New("timeout")

	repo := &mockUserRepo{
		getByIDFn: func(_ context.Context, _ uint) (*usersdomain.User, error) {
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
