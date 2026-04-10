package payments

import (
	stripeadapter "golang_boilerplate_module/internal/modules/payments/infra/stripe"

	"go.uber.org/fx"
)

var Module = fx.Module("payments",
	fx.Provide(stripeadapter.NewStripeGateway),
)
