package usecases_test

import (
	"context"
	"errors"
	"testing"

	"golang_boilerplate_module/internal/modules/users/application/usecases"
	"golang_boilerplate_module/internal/modules/users/domain"
	"golang_boilerplate_module/internal/shared/domain/exceptions"
)

func TestCreateUserUseCase_Success(t *testing.T) {
	repo := &mockUserRepo{

		getByEmailFn: func(_ context.Context, _ string) (*domain.User, error) {
			return nil, nil
		},
		addFn: func(_ context.Context, u *domain.User) (*domain.User, error) {
			u.ID = 1
			return u, nil
		},
	}

	uc := usecases.NewCreateUserUseCase(repo, &mockLogger{})
	out, err := uc.Execute(context.Background(), usecases.CreateUserInput{
		Name:  "João Silva",
		Email: "joao@example.com",
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if out.ID != 1 {
		t.Fatalf("expected ID=1, got %d", out.ID)
	}
	if out.Name != "João Silva" {
		t.Fatalf("expected name=João Silva, got %q", out.Name)
	}
	if out.Email != "joao@example.com" {
		t.Fatalf("expected email=joao@example.com, got %q", out.Email)
	}
}

func TestCreateUserUseCase_MissingName(t *testing.T) {
	uc := usecases.NewCreateUserUseCase(&mockUserRepo{}, &mockLogger{})

	_, err := uc.Execute(context.Background(), usecases.CreateUserInput{
		Name:  "",
		Email: "joao@example.com",
	})

	if err == nil {
		t.Fatal("expected error for missing name, got nil")
	}

	var domainErr *exceptions.DomainError
	if !errors.As(err, &domainErr) {
		t.Fatalf("expected DomainError, got %T", err)
	}
	if domainErr.Code != exceptions.CodeBadRequest {
		t.Fatalf("expected BAD_REQUEST, got %s", domainErr.Code)
	}
}

func TestCreateUserUseCase_MissingEmail(t *testing.T) {
	uc := usecases.NewCreateUserUseCase(&mockUserRepo{}, &mockLogger{})

	_, err := uc.Execute(context.Background(), usecases.CreateUserInput{
		Name:  "João",
		Email: "",
	})

	if err == nil {
		t.Fatal("expected error for missing email, got nil")
	}

	var domainErr *exceptions.DomainError
	if !errors.As(err, &domainErr) {
		t.Fatalf("expected DomainError, got %T", err)
	}
	if domainErr.Code != exceptions.CodeBadRequest {
		t.Fatalf("expected BAD_REQUEST, got %s", domainErr.Code)
	}
}

func TestCreateUserUseCase_DuplicateEmail(t *testing.T) {
	existing := &domain.User{ID: 99, Name: "Outro", Email: "dup@example.com"}

	repo := &mockUserRepo{
		getByEmailFn: func(_ context.Context, _ string) (*domain.User, error) {
			return existing, nil
		},
	}

	uc := usecases.NewCreateUserUseCase(repo, &mockLogger{})
	_, err := uc.Execute(context.Background(), usecases.CreateUserInput{
		Name:  "Novo",
		Email: "dup@example.com",
	})

	if err == nil {
		t.Fatal("expected error for duplicate email, got nil")
	}

	var domainErr *exceptions.DomainError
	if !errors.As(err, &domainErr) {
		t.Fatalf("expected DomainError, got %T", err)
	}
	if domainErr.Code != exceptions.CodeUnprocessable {
		t.Fatalf("expected UNPROCESSABLE, got %s", domainErr.Code)
	}
}

func TestCreateUserUseCase_RepositoryError(t *testing.T) {
	repoErr := errors.New("connection reset")

	repo := &mockUserRepo{
		getByEmailFn: func(_ context.Context, _ string) (*domain.User, error) {
			return nil, nil
		},
		addFn: func(_ context.Context, _ *domain.User) (*domain.User, error) {
			return nil, repoErr
		},
	}

	uc := usecases.NewCreateUserUseCase(repo, &mockLogger{})
	_, err := uc.Execute(context.Background(), usecases.CreateUserInput{
		Name:  "Teste",
		Email: "teste@example.com",
	})

	if err == nil {
		t.Fatal("expected error from repository, got nil")
	}
	if !errors.Is(err, repoErr) {
		t.Fatalf("expected wrapped repoErr, got %v", err)
	}
}
