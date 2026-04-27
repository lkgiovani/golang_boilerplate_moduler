package plans

import (
	"golang_boilerplate_module/internal/modules/plans/application/plansusecases"
	"golang_boilerplate_module/internal/modules/plans/infra/planshttp"
	"golang_boilerplate_module/internal/modules/plans/infra/planspersistence"

	"go.uber.org/fx"
)

var Module = fx.Module("plans",
	fx.Provide(
		// Repositories
		planspersistence.NewGORMPlanRepository,
		planspersistence.NewGORMSubscriptionRepository,
		planspersistence.NewGORMPaymentEventRepository,
		// Use cases
		plansusecases.NewCreatePlanUseCase,
		plansusecases.NewUpdatePlanUseCase,
		plansusecases.NewListPlansUseCase,
		plansusecases.NewGetPlanUseCase,
		plansusecases.NewDeletePlanUseCase,
		plansusecases.NewSubscribeUseCase,
		plansusecases.NewCancelSubscriptionUseCase,
		plansusecases.NewGetSubscriptionUseCase,
		plansusecases.NewHandleWebhookUseCase,
		plansusecases.NewUpdatePaymentMethodUseCase,
		plansusecases.NewRefundPaymentUseCase,
		plansusecases.NewCreateBillingPortalUseCase,
		plansusecases.NewGetSubscriptionStatusUseCase,
		// HTTP
		planshttp.NewPlansController,
		planshttp.NewWebhookController,
		planshttp.NewFeatureGateMiddleware,
	),
	fx.Invoke(planshttp.RegisterRoutes),
)
