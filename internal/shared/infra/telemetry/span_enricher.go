package telemetry

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// RequestIDContextKey is the context key used to store the request ID.
type requestIDContextKey struct{}

var RequestIDContextKey = requestIDContextKey{}

// SpanEnricher is a custom SpanProcessor that injects the requestId from the
// Go context into every span as an attribute — equivalent to contextSpanProcessor.ts.
type SpanEnricher struct{}

func (s SpanEnricher) OnStart(parent context.Context, span sdktrace.ReadWriteSpan) {
	if reqID, ok := parent.Value(RequestIDContextKey).(string); ok && reqID != "" {
		span.SetAttributes(attribute.String("http.request.header.x_request_id", reqID))
	}
}

func (s SpanEnricher) OnEnd(_ sdktrace.ReadOnlySpan) {}

func (s SpanEnricher) Shutdown(_ context.Context) error { return nil }

func (s SpanEnricher) ForceFlush(_ context.Context) error { return nil }
