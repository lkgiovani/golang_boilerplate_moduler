package usersusecases_test

import (
	"context"
	"errors"
	"testing"

	"golang_boilerplate_module/internal/modules/users/application/usersusecases"
	"golang_boilerplate_module/internal/modules/users/usersdomain"
	"golang_boilerplate_module/internal/shared/domain/exceptions"
)

func TestCreateUserUseCase_Success(t *testing.T) {
	repo := &mockUserRepo{
		getByEmailFn: func(_ context.Context, _ string) (*usersdomain.User, error) {
			return nil, nil
		},
		addFn: func(_ context.Context, u *usersdomain.User) (*usersdomain.User, error) {
			u.ID = 1
			return u, nil
		},
	}

	uc := usersusecases.NewCreateUserUseCase(repo, &mockLogger{})
	out, err := uc.Execute(context.Background(), usersusecases.CreateUserInput{
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
	uc := usersusecases.NewCreateUserUseCase(&mockUserRepo{}, &mockLogger{})

	_, err := uc.Execute(context.Background(), usersusecases.CreateUserInput{
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
	uc := usersusecases.NewCreateUserUseCase(&mockUserRepo{}, &mockLogger{})

	_, err := uc.Execute(context.Background(), usersusecases.CreateUserInput{
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
	existing := &usersdomain.User{ID: 99, Name: "Outro", Email: "dup@example.com"}

	repo := &mockUserRepo{
		getByEmailFn: func(_ context.Context, _ string) (*usersdomain.User, error) {
			return existing, nil
		},
	}

	uc := usersusecases.NewCreateUserUseCase(repo, &mockLogger{})
	_, err := uc.Execute(context.Background(), usersusecases.CreateUserInput{
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
		getByEmailFn: func(_ context.Context, _ string) (*usersdomain.User, error) {
			return nil, nil
		},
		addFn: func(_ context.Context, _ *usersdomain.User) (*usersdomain.User, error) {
			return nil, repoErr
		},
	}

	uc := usersusecases.NewCreateUserUseCase(repo, &mockLogger{})
	_, err := uc.Execute(context.Background(), usersusecases.CreateUserInput{
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
