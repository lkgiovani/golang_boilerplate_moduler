package users

import (
	"golang_boilerplate_module/internal/modules/users/application/usersusecases"
	"golang_boilerplate_module/internal/modules/users/infra/usershttp"
	"golang_boilerplate_module/internal/modules/users/infra/userspersistence"

	"go.uber.org/fx"
)

var Module = fx.Module("users",
	fx.Provide(
		userspersistence.NewGORMUserRepository,
		usersusecases.NewCreateUserUseCase,
		usersusecases.NewGetUserUseCase,
		usershttp.NewUserController,
	),
	fx.Invoke(usershttp.RegisterRoutes),
)
