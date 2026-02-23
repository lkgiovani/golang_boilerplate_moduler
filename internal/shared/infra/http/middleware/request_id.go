package middleware

import (
	"context"

	"golang_boilerplate_module/internal/shared/domain/providers"
	"golang_boilerplate_module/internal/shared/infra/telemetry"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	oteltrace "go.opentelemetry.io/otel/trace"
)

const loggerLocalsKey = "logger"

func RequestID(rootLogger providers.LoggerProvider) fiber.Handler {
	return func(c *fiber.Ctx) error {
		reqID := c.Get("X-Request-ID")
		if reqID == "" {
			reqID = uuid.NewString()
		}

		fields := []any{"requestId", reqID}

		ctx := c.UserContext()
		span := oteltrace.SpanFromContext(ctx)
		if spanCtx := span.SpanContext(); spanCtx.IsValid() {
			fields = append(fields,
				"traceId", spanCtx.TraceID().String(),
				"spanId", spanCtx.SpanID().String(),
			)
		}

		requestLogger := rootLogger.With(fields...)
		c.Locals(loggerLocalsKey, requestLogger)

		ctx = context.WithValue(ctx, telemetry.RequestIDContextKey, reqID)
		c.SetUserContext(ctx)

		c.Set("X-Request-ID", reqID)
		return c.Next()
	}
}

func LoggerFromLocals(c *fiber.Ctx, fallback providers.LoggerProvider) providers.LoggerProvider {
	if l, ok := c.Locals(loggerLocalsKey).(providers.LoggerProvider); ok {
		return l
	}
	return fallback
}