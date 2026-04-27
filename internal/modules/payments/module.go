package payments

import (
	"fmt"

	"golang_boilerplate_module/internal/config"
	stripeadapter "golang_boilerplate_module/internal/modules/payments/infra/stripe"
	"golang_boilerplate_module/internal/shared/domain/providers"

	"go.uber.org/fx"
)

// gatewayFactory constructs an adapter from the shared Config.
type gatewayFactory func(*config.Config) providers.PaymentGateway

// registry holds the set of known adapters. Adding a provider = one line + new infra package.
var registry = map[string]gatewayFactory{
	"stripe": stripeadapter.NewStripeGateway,
}

// providePaymentGateway reads cfg.PaymentGateway.Name and returns the matching adapter.
// Unknown names fail fast (fatal fx startup).
func providePaymentGateway(cfg *config.Config) (providers.PaymentGateway, error) {
	name := cfg.PaymentGateway.Name
	factory, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("payment gateway %q not registered (registered: %v)", name, registeredNames())
	}
	return factory(cfg), nil
}

func registeredNames() []string {
	names := make([]string, 0, len(registry))
	for k := range registry {
		names = append(names, k)
	}
	return names
}

var Module = fx.Module("payments",
	fx.Provide(providePaymentGateway),
)
