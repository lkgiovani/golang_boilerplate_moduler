package bootstrap

import (
	"context"
	"fmt"

	"golang_boilerplate_module/internal/config"
	"golang_boilerplate_module/internal/modules/health"
	"golang_boilerplate_module/internal/modules/users"
	"golang_boilerplate_module/internal/shared/domain/providers"
	sharedfx "golang_boilerplate_module/internal/shared/infra"
	"golang_boilerplate_module/internal/shared/infra/http/middleware"
	"golang_boilerplate_module/internal/shared/infra/persistence"

	otelfiber "github.com/gofiber/contrib/otelfiber/v2"
	"github.com/gofiber/fiber/v2"
	fibercors "github.com/gofiber/fiber/v2/middleware/cors"
	"go.uber.org/fx"
	"gorm.io/gorm"
)

func NewFiberApp(logger providers.LoggerProvider) *fiber.App {
	app := fiber.New(fiber.Config{
		ErrorHandler: middleware.NewErrorHandler(logger),
	})

	app.Use(fibercors.New())
	app.Use(otelfiber.Middleware())
	app.Use(middleware.HTTPMetrics())
	app.Use(middleware.RequestID(logger))

	return app
}

func StartFiberApp(
	lc fx.Lifecycle,
	app *fiber.App,
	cfg *config.Config,
	logger providers.LoggerProvider,
	db *gorm.DB,
) {
	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			addr := fmt.Sprintf(":%d", cfg.App.Port)
			logger.Info("Server starting",
				"addr", addr,
				"env", cfg.App.Env,
				"version", cfg.App.Version,
			)
			go func() {
				if err := app.Listen(addr); err != nil {
					logger.Error("Server stopped with error", "error", err)
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("Shutting down gracefully...")
			_ = app.ShutdownWithContext(ctx)
			_ = persistence.CloseDB(db)
			_ = logger.Sync()
			return nil
		},
	})
}

var App = fx.Options(
	fx.Provide(config.NewConfig),
	fx.Provide(NewFiberApp),
	sharedfx.Module,
	health.Module,
	users.Module,
	fx.Invoke(StartFiberApp),
)
