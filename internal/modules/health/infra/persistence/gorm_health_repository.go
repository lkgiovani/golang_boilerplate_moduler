package persistence

import (
	"context"

	healthrepo "golang_boilerplate_module/internal/modules/health/domain/repositories"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"gorm.io/gorm"
)

var dbTracer = otel.Tracer("health.persistence")

type GORMHealthRepository struct {
	db *gorm.DB
}

// NewGORMHealthRepository is the fx constructor.
// Returns the HealthRepository interface, not the concrete type.
func NewGORMHealthRepository(db *gorm.DB) healthrepo.HealthRepository {
	return &GORMHealthRepository{db: db}
}

func (r *GORMHealthRepository) Ping(ctx context.Context) (bool, error) {
	ctx, span := dbTracer.Start(ctx, "GORMHealthRepository.Ping")
	defer span.End()

	span.SetAttributes(attribute.String("db.operation", "ping"))

	sqlDB, err := r.db.DB()
	if err != nil {
		span.SetStatus(codes.Error, "failed to get sql.DB")
		span.RecordError(err)
		return false, nil
	}

	if err := sqlDB.PingContext(ctx); err != nil {
		span.SetStatus(codes.Error, "ping failed")
		span.RecordError(err)
		return false, nil
	}

	span.SetStatus(codes.Ok, "ping ok")
	return true, nil
}
