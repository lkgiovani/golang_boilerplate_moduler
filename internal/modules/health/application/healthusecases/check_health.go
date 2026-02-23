package healthusecases

import (
	"context"

	"golang_boilerplate_module/internal/modules/health/healthdomain"
	"golang_boilerplate_module/internal/shared/domain/providers"
	"golang_boilerplate_module/internal/shared/infra/observability"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

type CheckHealthOutput struct {
	Status healthdomain.HealthStatus `json:"status"`
}

type CheckHealthUseCase struct {
	logger providers.LoggerProvider
}

func NewCheckHealthUseCase(logger providers.LoggerProvider) *CheckHealthUseCase {
	return &CheckHealthUseCase{logger: logger}
}

func (uc *CheckHealthUseCase) Execute(ctx context.Context) CheckHealthOutput {
	ctx, span := healthTracer.Start(ctx, "CheckHealthUseCase.Execute")
	defer span.End()

	log := observability.LoggerWithTrace(ctx, uc.logger).With("usecase", "CheckHealth")

	output := CheckHealthOutput{Status: healthdomain.HealthStatusHealthy}

	span.SetAttributes(attribute.String("health.status", string(output.Status)))
	span.SetStatus(codes.Ok, "liveness ok")
	log.Debug("liveness check called", "status", output.Status)

	return output
}
