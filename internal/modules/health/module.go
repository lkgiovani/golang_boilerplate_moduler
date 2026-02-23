package health

import (
	"golang_boilerplate_module/internal/modules/health/application/usecases"
	healthhttp "golang_boilerplate_module/internal/modules/health/infra/http"
	healthpersistence "golang_boilerplate_module/internal/modules/health/infra/persistence"

	"go.uber.org/fx"
)

var Module = fx.Module("health",
	fx.Provide(
		healthpersistence.NewGORMHealthRepository,
		usecases.NewCheckHealthUseCase,
		usecases.NewCheckReadinessUseCase,
		healthhttp.NewHealthController,
	),
	fx.Invoke(healthhttp.RegisterRoutes),
)
