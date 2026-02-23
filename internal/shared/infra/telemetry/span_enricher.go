package telemetry

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

type requestIDContextKey struct{}

var RequestIDContextKey = requestIDContextKey{}

type SpanEnricher struct{}

func (s SpanEnricher) OnStart(parent context.Context, span sdktrace.ReadWriteSpan) {
	if reqID, ok := parent.Value(RequestIDContextKey).(string); ok && reqID != "" {
		span.SetAttributes(attribute.String("http.request.header.x_request_id", reqID))
	}
}

func (s SpanEnricher) OnEnd(_ sdktrace.ReadOnlySpan) {}

func (s SpanEnricher) Shutdown(_ context.Context) error { return nil }

func (s SpanEnricher) ForceFlush(_ context.Context) error { return nil }