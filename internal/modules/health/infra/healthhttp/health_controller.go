package healthhttp

import (
	"golang_boilerplate_module/internal/modules/health/application/healthusecases"
	"golang_boilerplate_module/internal/shared/domain/providers"
	"golang_boilerplate_module/internal/shared/infra/http/middleware"
	"golang_boilerplate_module/internal/shared/infra/observability"

	"github.com/gofiber/fiber/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

var tracer = otel.Tracer("health.http")

type HealthController struct {
	checkHealth    *healthusecases.CheckHealthUseCase
	checkReadiness *healthusecases.CheckReadinessUseCase
	logger         providers.LoggerProvider
}

func NewHealthController(
	checkHealth *healthusecases.CheckHealthUseCase,
	checkReadiness *healthusecases.CheckReadinessUseCase,
	logger providers.LoggerProvider,
) *HealthController {
	return &HealthController{
		checkHealth:    checkHealth,
		checkReadiness: checkReadiness,
		logger:         logger,
	}
}

func (h *HealthController) CheckHealth(c *fiber.Ctx) error {
	ctx, span := tracer.Start(c.UserContext(), "HealthController.CheckHealth")
	defer span.End()

	log := middleware.LoggerFromLocals(c, h.logger).With("handler", "HealthController.CheckHealth")

	output := h.checkHealth.Execute(ctx)

	span.SetAttributes(
		attribute.String("health.status", string(output.Status)),
		attribute.Int("http.response.status_code", 200),
	)
	span.SetStatus(codes.Ok, "liveness ok")
	log.Debug("liveness probe responded", "status", output.Status)

	return c.JSON(output)
}

func (h *HealthController) CheckReadiness(c *fiber.Ctx) error {
	ctx, span := tracer.Start(c.UserContext(), "HealthController.CheckReadiness")
	defer span.End()

	log := middleware.LoggerFromLocals(c, h.logger).With("handler", "HealthController.CheckReadiness")

	output, err := h.checkReadiness.Execute(ctx)
	if err != nil {
		log.Error("readiness probe failed", "error", err.Error())
		observability.RecordError(span, err)
		return err
	}

	span.SetAttributes(
		attribute.String("health.status", string(output.Status)),
		attribute.Int("http.response.status_code", 200),
	)
	span.SetStatus(codes.Ok, "readiness ok")
	log.Info("readiness probe responded", "status", output.Status)

	return c.JSON(output)
}