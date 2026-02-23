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

var domainToHTTPStatus = map[exceptions.ExceptionCode]int{
	exceptions.CodeBadRequest:         400,
	exceptions.CodeUnauthorized:       401,
	exceptions.CodeForbidden:          403,
	exceptions.CodeNotFound:           404,
	exceptions.CodeUnprocessable:      422,
	exceptions.CodeInternal:           500,
	exceptions.CodeServiceUnavailable: 503,
}

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