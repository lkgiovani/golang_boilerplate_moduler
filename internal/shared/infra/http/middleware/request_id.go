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

// RequestID is a Fiber middleware that:
// 1. Reads or generates X-Request-ID header
// 2. Extracts traceId/spanId from the active OTel span
// 3. Creates a request-scoped child logger with those fields
// 4. Stores the logger in c.Locals and the requestId in c.UserContext
//
// Equivalent to requestContext.ts + PinoRequestLoggerProvider in the TS project.
func RequestID(rootLogger providers.LoggerProvider) fiber.Handler {
	return func(c *fiber.Ctx) error {
		reqID := c.Get("X-Request-ID")
		if reqID == "" {
			reqID = uuid.NewString()
		}

		// Build log fields — always include requestId
		fields := []any{"requestId", reqID}

		// Extract traceId/spanId from the active OTel span (created by otelfiber before this middleware)
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

		// Store requestId in Go context so SpanEnricher can pick it up
		ctx = context.WithValue(ctx, telemetry.RequestIDContextKey, reqID)
		c.SetUserContext(ctx)

		c.Set("X-Request-ID", reqID)
		return c.Next()
	}
}

// LoggerFromContext retrieves the request-scoped logger stored by RequestID middleware.
// Falls back to rootLogger if not available (e.g. background goroutines).
func LoggerFromLocals(c *fiber.Ctx, fallback providers.LoggerProvider) providers.LoggerProvider {
	if l, ok := c.Locals(loggerLocalsKey).(providers.LoggerProvider); ok {
		return l
	}
	return fallback
}
