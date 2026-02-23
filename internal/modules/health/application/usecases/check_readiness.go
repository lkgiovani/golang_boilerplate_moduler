package usecases

import (
	"context"

	"golang_boilerplate_module/internal/modules/health/domain"
	healthrepo "golang_boilerplate_module/internal/modules/health/domain/repositories"
	"golang_boilerplate_module/internal/shared/domain/exceptions"
	"golang_boilerplate_module/internal/shared/domain/providers"
	"golang_boilerplate_module/internal/shared/infra/observability"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

var healthTracer = otel.Tracer("health")

type ComponentHealth struct {
	Status domain.HealthStatus `json:"status"`
}

type CheckReadinessOutput struct {
	Status     domain.HealthStatus        `json:"status"`
	Components map[string]ComponentHealth `json:"components"`
}

type CheckReadinessUseCase struct {
	healthRepo healthrepo.HealthRepository
	logger     providers.LoggerProvider
}

func NewCheckReadinessUseCase(healthRepo healthrepo.HealthRepository, logger providers.LoggerProvider) *CheckReadinessUseCase {
	return &CheckReadinessUseCase{healthRepo: healthRepo, logger: logger}
}

func (uc *CheckReadinessUseCase) Execute(ctx context.Context) (CheckReadinessOutput, error) {
	ctx, span := healthTracer.Start(ctx, "CheckReadinessUseCase.Execute")
	defer span.End()

	log := observability.LoggerWithTrace(ctx, uc.logger).With("usecase", "CheckReadiness")

	dbPing, _ := uc.healthRepo.Ping(ctx)

	dbStatus := domain.ToHealthStatus(dbPing)
	span.SetAttributes(attribute.String("health.database.status", string(dbStatus)))

	components := map[string]ComponentHealth{
		"database": {Status: dbStatus},
	}

	for name, component := range components {
		if component.Status == domain.HealthStatusUnhealthy {
			err := exceptions.NewServiceUnavailableException(
				"Readiness check detected unhealthy components",
				map[string]any{"component": name},
			)
			log.Error("readiness check failed — unhealthy component",
				"component", name,
				"status", component.Status,
			)
			observability.RecordError(span, err)
			return CheckReadinessOutput{}, err
		}
	}

	span.SetStatus(codes.Ok, "all components healthy")
	log.Info("readiness check passed", "components", len(components))

	return CheckReadinessOutput{
		Status:     domain.HealthStatusHealthy,
		Components: components,
	}, nil
}
