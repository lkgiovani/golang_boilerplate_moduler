package healthpersistence

import (
	"context"

	"golang_boilerplate_module/internal/modules/health/healthdomain/healthrepo"

	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"gorm.io/gorm"
)

var dbTracer = otel.Tracer("health.persistence")

type GORMHealthRepository struct {
	db    *gorm.DB
	redis *redis.Client
}

func NewGORMHealthRepository(db *gorm.DB, redisClient *redis.Client) healthrepo.HealthRepository {
	return &GORMHealthRepository{db: db, redis: redisClient}
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

func (r *GORMHealthRepository) PingRedis(ctx context.Context) bool {
	ctx, span := dbTracer.Start(ctx, "GORMHealthRepository.PingRedis")
	defer span.End()

	span.SetAttributes(attribute.String("db.operation", "redis_ping"))

	if err := r.redis.Ping(ctx).Err(); err != nil {
		span.SetStatus(codes.Error, "redis ping failed")
		span.RecordError(err)
		return false
	}

	span.SetStatus(codes.Ok, "redis ping ok")
	return true
}
