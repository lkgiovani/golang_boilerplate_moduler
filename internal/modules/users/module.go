package users

import (
	"golang_boilerplate_module/internal/modules/users/application/usecases"
	usershttp "golang_boilerplate_module/internal/modules/users/infra/http"
	userspersistence "golang_boilerplate_module/internal/modules/users/infra/persistence"

	"go.uber.org/fx"
)

var Module = fx.Module("users",
	fx.Provide(
		userspersistence.NewGORMUserRepository,
		usecases.NewCreateUserUseCase,
		usecases.NewGetUserUseCase,
		usershttp.NewUserController,
	),
	fx.Invoke(usershttp.RegisterRoutes),
)
