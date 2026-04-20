package observability

import (
	"errors"
	"fmt"
	"testing"

	"golang_boilerplate_module/internal/shared/domain/errs"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/embedded"
)

// recordingSpan is a minimal trace.Span mock that records attribute, status,
// and RecordError calls so tests can assert on them.
type recordingSpan struct {
	embedded.Span
	recording      bool
	attributes     []attribute.KeyValue
	statusCode     codes.Code
	statusDesc     string
	recordedErrors []error
}

func (s *recordingSpan) End(options ...trace.SpanEndOption) {}
func (s *recordingSpan) AddEvent(name string, options ...trace.EventOption) {
}
func (s *recordingSpan) AddLink(link trace.Link) {}
func (s *recordingSpan) IsRecording() bool       { return s.recording }
func (s *recordingSpan) RecordError(err error, options ...trace.EventOption) {
	s.recordedErrors = append(s.recordedErrors, err)
}
func (s *recordingSpan) SpanContext() trace.SpanContext { return trace.SpanContext{} }
func (s *recordingSpan) SetStatus(code codes.Code, description string) {
	s.statusCode = code
	s.statusDesc = description
}
func (s *recordingSpan) SetName(name string) {}
func (s *recordingSpan) SetAttributes(kv ...attribute.KeyValue) {
	s.attributes = append(s.attributes, kv...)
}
func (s *recordingSpan) TracerProvider() trace.TracerProvider { return nil }

// findAttr returns the value of the first attribute with the given key.
func findAttr(attrs []attribute.KeyValue, key string) (attribute.Value, bool) {
	for _, a := range attrs {
		if string(a.Key) == key {
			return a.Value, true
		}
	}
	return attribute.Value{}, false
}

func TestRecordError_Nil(t *testing.T) {
	span := &recordingSpan{recording: true}
	RecordError(span, nil)
	if len(span.attributes) != 0 || len(span.recordedErrors) != 0 {
		t.Fatalf("nil error should be a no-op, got attrs=%v errors=%v", span.attributes, span.recordedErrors)
	}
}

func TestRecordError_NonRecordingSpan(t *testing.T) {
	span := &recordingSpan{recording: false}
	RecordError(span, errs.Errorf(errs.EBADREQUEST, "x"))
	if len(span.attributes) != 0 || len(span.recordedErrors) != 0 {
		t.Fatalf("non-recording span should be a no-op, got attrs=%v errors=%v", span.attributes, span.recordedErrors)
	}
}

func TestRecordError_ErrsErrorUnauthorized(t *testing.T) {
	span := &recordingSpan{recording: true}
	err := errs.Errorf(errs.EUNAUTHORIZED, "invalid creds")
	RecordError(span, err)

	statusAttr, ok := findAttr(span.attributes, "http.response.status_code")
	if !ok || statusAttr.AsInt64() != 401 {
		t.Errorf("http.response.status_code = %v, want 401", statusAttr.AsInt64())
	}
	typeAttr, ok := findAttr(span.attributes, "error.type")
	if !ok || typeAttr.AsString() != "unauthorized" {
		t.Errorf("error.type = %q, want %q", typeAttr.AsString(), "unauthorized")
	}
	if span.statusCode != codes.Error {
		t.Errorf("statusCode = %v, want Error", span.statusCode)
	}
	if len(span.recordedErrors) != 1 {
		t.Errorf("recordedErrors len = %d, want 1", len(span.recordedErrors))
	}
}

func TestRecordError_ErrsErrorNotFound(t *testing.T) {
	span := &recordingSpan{recording: true}
	RecordError(span, errs.Errorf(errs.ENOTFOUND, "x"))
	statusAttr, _ := findAttr(span.attributes, "http.response.status_code")
	typeAttr, _ := findAttr(span.attributes, "error.type")
	if statusAttr.AsInt64() != 404 {
		t.Errorf("status = %d, want 404", statusAttr.AsInt64())
	}
	if typeAttr.AsString() != "not_found" {
		t.Errorf("error.type = %q, want %q", typeAttr.AsString(), "not_found")
	}
}

func TestRecordError_WrappedErrsError(t *testing.T) {
	span := &recordingSpan{recording: true}
	wrapped := fmt.Errorf("ctx: %w", errs.Errorf(errs.EFORBIDDEN, "x"))
	RecordError(span, wrapped)
	statusAttr, _ := findAttr(span.attributes, "http.response.status_code")
	typeAttr, _ := findAttr(span.attributes, "error.type")
	if statusAttr.AsInt64() != 403 {
		t.Errorf("status = %d, want 403", statusAttr.AsInt64())
	}
	if typeAttr.AsString() != "forbidden" {
		t.Errorf("error.type = %q, want %q", typeAttr.AsString(), "forbidden")
	}
}

func TestRecordError_UnknownError(t *testing.T) {
	span := &recordingSpan{recording: true}
	RecordError(span, errors.New("raw"))
	statusAttr, _ := findAttr(span.attributes, "http.response.status_code")
	typeAttr, _ := findAttr(span.attributes, "error.type")
	if statusAttr.AsInt64() != 500 {
		t.Errorf("status = %d, want 500", statusAttr.AsInt64())
	}
	if typeAttr.AsString() != "internal" {
		t.Errorf("error.type = %q, want %q", typeAttr.AsString(), "internal")
	}
	if span.statusCode != codes.Error {
		t.Errorf("statusCode = %v, want Error", span.statusCode)
	}
	if len(span.recordedErrors) != 1 {
		t.Errorf("recordedErrors len = %d, want 1", len(span.recordedErrors))
	}
}
