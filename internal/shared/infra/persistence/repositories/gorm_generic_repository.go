package repositories

import (
	"context"
	"fmt"

	"golang_boilerplate_module/internal/shared/infra/persistence"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"gorm.io/gorm"
)

var dbTracer = otel.Tracer("shared.persistence")

type GORMGenericRepository[T any, ID comparable] struct {
	db         *gorm.DB
	entityName string
}

func NewGORMGenericRepository[T any, ID comparable](db *gorm.DB) *GORMGenericRepository[T, ID] {
	var zero T
	return &GORMGenericRepository[T, ID]{
		db:         db,
		entityName: fmt.Sprintf("%T", zero),
	}
}

func (r *GORMGenericRepository[T, ID]) Add(ctx context.Context, entity *T) (*T, error) {
	ctx, span := dbTracer.Start(ctx, r.entityName+".Add")
	defer span.End()

	span.SetAttributes(
		attribute.String("db.operation", "INSERT"),
		attribute.String("db.model", r.entityName),
	)

	if err := r.db.WithContext(ctx).Create(entity).Error; err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
		return nil, persistence.FromGORM(err, r.entityName)
	}

	span.SetStatus(codes.Ok, "inserted")
	return entity, nil
}

func (r *GORMGenericRepository[T, ID]) GetByID(ctx context.Context, id ID) (*T, error) {
	ctx, span := dbTracer.Start(ctx, r.entityName+".GetByID")
	defer span.End()

	span.SetAttributes(
		attribute.String("db.operation", "SELECT"),
		attribute.String("db.model", r.entityName),
	)

	var entity T
	if err := r.db.WithContext(ctx).First(&entity, "id = ?", id).Error; err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
		return nil, persistence.FromGORM(err, r.entityName)
	}

	span.SetStatus(codes.Ok, "found")
	return &entity, nil
}

func (r *GORMGenericRepository[T, ID]) UpdateByID(ctx context.Context, id ID, updates map[string]any) (*T, error) {
	ctx, span := dbTracer.Start(ctx, r.entityName+".UpdateByID")
	defer span.End()

	span.SetAttributes(
		attribute.String("db.operation", "UPDATE"),
		attribute.String("db.model", r.entityName),
	)

	var entity T
	if err := r.db.WithContext(ctx).First(&entity, "id = ?", id).Error; err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
		return nil, persistence.FromGORM(err, r.entityName)
	}
	if err := r.db.WithContext(ctx).Model(&entity).Updates(updates).Error; err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
		return nil, persistence.FromGORM(err, r.entityName)
	}

	span.SetStatus(codes.Ok, "updated")
	return &entity, nil
}

func (r *GORMGenericRepository[T, ID]) DeleteByID(ctx context.Context, id ID) error {
	ctx, span := dbTracer.Start(ctx, r.entityName+".DeleteByID")
	defer span.End()

	span.SetAttributes(
		attribute.String("db.operation", "DELETE"),
		attribute.String("db.model", r.entityName),
	)

	var entity T
	if err := r.db.WithContext(ctx).Delete(&entity, "id = ?", id).Error; err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
		return persistence.FromGORM(err, r.entityName)
	}

	span.SetStatus(codes.Ok, "deleted")
	return nil
}

func (r *GORMGenericRepository[T, ID]) DeleteAll(ctx context.Context) error {
	ctx, span := dbTracer.Start(ctx, r.entityName+".DeleteAll")
	defer span.End()

	span.SetAttributes(
		attribute.String("db.operation", "DELETE_ALL"),
		attribute.String("db.model", r.entityName),
	)

	var entity T
	if err := r.db.WithContext(ctx).Where("1 = 1").Delete(&entity).Error; err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
		return persistence.FromGORM(err, r.entityName)
	}

	span.SetStatus(codes.Ok, "deleted all")
	return nil
}
