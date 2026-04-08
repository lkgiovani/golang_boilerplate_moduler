package infra

import (
	"context"

	"golang_boilerplate_module/internal/config"
	"golang_boilerplate_module/internal/shared/domain/providers"
	"golang_boilerplate_module/internal/shared/infra/persistence"
	"golang_boilerplate_module/internal/shared/infra/providers/cache"
	"golang_boilerplate_module/internal/shared/infra/providers/email"
	zaplogger "golang_boilerplate_module/internal/shared/infra/providers/logger"
	"golang_boilerplate_module/internal/shared/infra/telemetry"

	"go.uber.org/fx"
	"gorm.io/gorm"
)

var Module = fx.Module("shared",
	fx.Provide(
		persistence.NewDB,
		persistence.NewRedisClient,
		fx.Annotate(
			zaplogger.NewZapLoggerProvider,
			fx.As(new(providers.LoggerProvider)),
		),
		fx.Annotate(
			cache.NewRedisCacheProvider,
			fx.As(new(providers.CacheProvider)),
		),
		fx.Annotate(
			email.NewSMTPEmailProvider,
			fx.As(new(providers.EmailProvider)),
		),
	),
	fx.Invoke(registerOTELLifecycle),
	fx.Invoke(persistence.RegisterRedisLifecycle),
)

func registerOTELLifecycle(lc fx.Lifecycle, cfg *config.Config, logger providers.LoggerProvider, db *gorm.DB) {
	var shutdownFn telemetry.ShutdownFunc

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			fn, err := telemetry.SetupOTel(cfg)
			if err != nil {
				return err
			}
			shutdownFn = fn
			logger.Info("OpenTelemetry initialized", "endpoint", cfg.Otel.Endpoint)
			return nil
		},
		OnStop: func(ctx context.Context) error {
			if shutdownFn != nil {
				return shutdownFn(ctx)
			}
			return nil
		},
	})
}