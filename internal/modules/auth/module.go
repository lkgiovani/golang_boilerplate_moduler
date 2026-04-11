package auth

import (
	"golang_boilerplate_module/internal/modules/auth/application/authusecases"
	"golang_boilerplate_module/internal/modules/auth/infra/authhttp"
	"golang_boilerplate_module/internal/modules/auth/infra/authpersistence"
	"golang_boilerplate_module/internal/modules/auth/infra/authproviders"

	"go.uber.org/fx"
)

var Module = fx.Module("auth",
	fx.Provide(
		authpersistence.NewGORMRefreshTokenRepository,
		authpersistence.NewGORMVerificationTokenRepository,
		authproviders.NewJWTTokenProvider,
		authusecases.NewVerificationSender,
		authusecases.NewRegisterUseCase,
		authusecases.NewLoginUseCase,
		authusecases.NewRefreshTokenUseCase,
		authusecases.NewLogoutUseCase,
		authusecases.NewConfirmEmailUseCase,
		authusecases.NewForgotPasswordUseCase,
		authusecases.NewResetPasswordUseCase,
		authusecases.NewChangePasswordUseCase,
		authhttp.NewAuthController,
		authhttp.NewAuthMiddleware,
	),
	fx.Invoke(authhttp.RegisterRoutes),
)
