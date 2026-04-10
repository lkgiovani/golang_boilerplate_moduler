package security

import (
	"golang_boilerplate_module/internal/modules/security/application/securityusecases"
	"golang_boilerplate_module/internal/modules/security/infra/securitypersistence"
	"golang_boilerplate_module/internal/modules/security/securityprovider"

	"go.uber.org/fx"
)

var Module = fx.Module("security",
	fx.Provide(
		securitypersistence.NewGORMSecurityRepository,
		securityusecases.DefaultPolicy,
		securityusecases.NewRecordActivityUseCase,
		securityusecases.NewCheckBlockUseCase,
		securityusecases.NewUnblockUserUseCase,
		securityprovider.NewBlockChecker,
	),
)
