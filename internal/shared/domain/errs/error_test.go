package errs

import (
	"errors"
	"fmt"
	"io"
	"testing"
)

func TestError_Error(t *testing.T) {
	e := &Error{Code: EBADREQUEST, Message: "boom"}
	if got, want := e.Error(), "[bad_request] boom"; got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}
}

func TestError_Unwrap(t *testing.T) {
	wrapped := io.EOF
	e := &Error{Code: EINTERNAL, Cause: wrapped}
	if got := e.Unwrap(); got != wrapped {
		t.Fatalf("Unwrap() = %v, want %v", got, wrapped)
	}

	empty := &Error{Code: EINTERNAL}
	if got := empty.Unwrap(); got != nil {
		t.Fatalf("Unwrap() with nil Cause = %v, want nil", got)
	}
}

func TestError_Is_MatchesByCode(t *testing.T) {
	a := &Error{Code: EUNAUTHORIZED, Message: "one"}
	b := &Error{Code: EUNAUTHORIZED, Message: "two"}
	if !errors.Is(a, b) {
		t.Fatalf("errors.Is(a, b) = false, want true (same Code, different pointers)")
	}
}

func TestError_Is_DifferentCode(t *testing.T) {
	a := &Error{Code: EBADREQUEST}
	b := &Error{Code: ENOTFOUND}
	if errors.Is(a, b) {
		t.Fatalf("errors.Is(a, b) = true, want false (different Code)")
	}
}

func TestError_Is_NonErrsTarget(t *testing.T) {
	e := &Error{Code: EINTERNAL}
	if errors.Is(e, io.EOF) {
		t.Fatalf("errors.Is(errsError, io.EOF) = true, want false")
	}
}

func TestError_Is_WalksUnwrapChain(t *testing.T) {
	inner := &Error{Code: EEXPIRED}
	wrapped := fmt.Errorf("ctx: %w", inner)
	target := &Error{Code: EEXPIRED}
	if !errors.Is(wrapped, target) {
		t.Fatalf("errors.Is(wrapped, target) = false, want true")
	}
}
