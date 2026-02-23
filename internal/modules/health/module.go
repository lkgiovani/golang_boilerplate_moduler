package health

import (
	"golang_boilerplate_module/internal/modules/health/application/healthusecases"
	"golang_boilerplate_module/internal/modules/health/infra/healthhttp"
	"golang_boilerplate_module/internal/modules/health/infra/healthpersistence"

	"go.uber.org/fx"
)

var Module = fx.Module("health",
	fx.Provide(
		healthpersistence.NewGORMHealthRepository,
		healthusecases.NewCheckHealthUseCase,
		healthusecases.NewCheckReadinessUseCase,
		healthhttp.NewHealthController,
	),
	fx.Invoke(healthhttp.RegisterRoutes),
)
