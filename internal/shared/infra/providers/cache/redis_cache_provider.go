package cache

import (
	"context"
	"time"

	"golang_boilerplate_module/internal/shared/domain/providers"

	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

var cacheTracer = otel.Tracer("cache.provider")

type RedisCacheProvider struct {
	client *redis.Client
}

func NewRedisCacheProvider(client *redis.Client) providers.CacheProvider {
	return &RedisCacheProvider{client: client}
}

func (r *RedisCacheProvider) Get(ctx context.Context, key string) (string, error) {
	ctx, span := cacheTracer.Start(ctx, "RedisCacheProvider.Get")
	defer span.End()

	span.SetAttributes(attribute.String("cache.key", key))

	val, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			span.SetAttributes(attribute.Bool("cache.hit", false))
			span.SetStatus(codes.Ok, "cache miss")
			return "", redis.Nil
		}
		span.SetStatus(codes.Error, "get failed")
		span.RecordError(err)
		return "", err
	}

	span.SetAttributes(attribute.Bool("cache.hit", true))
	span.SetStatus(codes.Ok, "cache hit")
	return val, nil
}

func (r *RedisCacheProvider) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	ctx, span := cacheTracer.Start(ctx, "RedisCacheProvider.Set")
	defer span.End()

	span.SetAttributes(
		attribute.String("cache.key", key),
		attribute.Int64("cache.ttl_ms", ttl.Milliseconds()),
	)

	if err := r.client.Set(ctx, key, value, ttl).Err(); err != nil {
		span.SetStatus(codes.Error, "set failed")
		span.RecordError(err)
		return err
	}

	span.SetStatus(codes.Ok, "set ok")
	return nil
}

func (r *RedisCacheProvider) Delete(ctx context.Context, key string) error {
	ctx, span := cacheTracer.Start(ctx, "RedisCacheProvider.Delete")
	defer span.End()

	span.SetAttributes(attribute.String("cache.key", key))

	if err := r.client.Del(ctx, key).Err(); err != nil {
		span.SetStatus(codes.Error, "delete failed")
		span.RecordError(err)
		return err
	}

	span.SetStatus(codes.Ok, "delete ok")
	return nil
}

func (r *RedisCacheProvider) Exists(ctx context.Context, key string) (bool, error) {
	ctx, span := cacheTracer.Start(ctx, "RedisCacheProvider.Exists")
	defer span.End()

	span.SetAttributes(attribute.String("cache.key", key))

	count, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		span.SetStatus(codes.Error, "exists failed")
		span.RecordError(err)
		return false, err
	}

	exists := count > 0
	span.SetAttributes(attribute.Bool("cache.exists", exists))
	span.SetStatus(codes.Ok, "exists ok")
	return exists, nil
}
