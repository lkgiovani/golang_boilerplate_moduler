package observability

import (
	"context"

	"golang_boilerplate_module/internal/shared/domain/errs"
	"golang_boilerplate_module/internal/shared/domain/providers"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// RecordError records an error on the given span with standard attributes
// derived from the errs.Code. Unknown errors are recorded as EINTERNAL/500.
//
// Attributes set:
//
//	http.response.status_code — from errs.HTTPStatus(code)
//	error.type                 — the lowercase string value of the code
//
// This function is the single integration point between domain errors
// and OTel. It intentionally does NOT read Metadata (T-8-02) and does NOT
// log the Cause chain — only the top-level Message drives span.SetStatus.
func RecordError(span oteltrace.Span, err error) {
	if err == nil || !span.IsRecording() {
		return
	}

	code := errs.ErrorCode(err)
	if code == "" {
		code = errs.EINTERNAL
	}
	status := errs.HTTPStatus(code)
	msg := errs.ErrorMessage(err)

	span.SetAttributes(
		attribute.Int("http.response.status_code", status),
		attribute.String("error.type", string(code)),
	)
	span.SetStatus(codes.Error, msg)
	span.RecordError(err)
}

// LoggerWithTrace returns a logger enriched with trace and span IDs
// from the current context. Unchanged from the pre-Phase-8 behavior.
func LoggerWithTrace(ctx context.Context, logger providers.LoggerProvider) providers.LoggerProvider {
	span := oteltrace.SpanFromContext(ctx)
	if spanCtx := span.SpanContext(); spanCtx.IsValid() {
		return logger.With(
			"traceId", spanCtx.TraceID().String(),
			"spanId", spanCtx.SpanID().String(),
		)
	}
	return logger
}
