package observability

import (
	"context"
	"errors"

	"golang_boilerplate_module/internal/shared/domain/exceptions"
	"golang_boilerplate_module/internal/shared/domain/providers"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// domainToHTTPStatus maps exception codes to HTTP status codes —
// mirrors the same mapping used by the error handler middleware.
var domainToHTTPStatus = map[exceptions.ExceptionCode]int{
	exceptions.CodeBadRequest:         400,
	exceptions.CodeUnauthorized:       401,
	exceptions.CodeForbidden:          403,
	exceptions.CodeNotFound:           404,
	exceptions.CodeUnprocessable:      422,
	exceptions.CodeInternal:           500,
	exceptions.CodeServiceUnavailable: 503,
}

// RecordError annotates the span with:
//   - http.response.status_code  (derived from DomainError.Code)
//   - error.type                 (the ExceptionCode string, e.g. "UNPROCESSABLE")
//   - span status = Error with the error message
//
// Works with any error; falls back to 500 for non-domain errors.
func RecordError(span oteltrace.Span, err error) {
	if err == nil || !span.IsRecording() {
		return
	}

	var domainErr *exceptions.DomainError
	if errors.As(err, &domainErr) {
		status, ok := domainToHTTPStatus[domainErr.Code]
		if !ok {
			status = 500
		}
		span.SetAttributes(
			attribute.Int("http.response.status_code", status),
			attribute.String("error.type", string(domainErr.Code)),
		)
		span.SetStatus(codes.Error, domainErr.Message)
		span.RecordError(err)
	} else {
		span.SetAttributes(
			attribute.Int("http.response.status_code", 500),
			attribute.String("error.type", "INTERNAL"),
		)
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
	}
}

// LoggerWithTrace enriches the given logger with traceId and spanId
// extracted from the active OTel span in ctx.
//
// Call this at the top of every Execute() so that all log lines —
// use cases, repositories — carry the same traceId visible in Tempo
// and correlated in Loki. No interface change or context storage needed:
// the span is already in ctx because the controller starts it first.
//
// Usage:
//
//	log := observability.LoggerWithTrace(ctx, uc.logger).With("usecase", "CreateUser")
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
