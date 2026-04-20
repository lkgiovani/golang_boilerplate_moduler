package errs

import (
	"errors"
	"fmt"
	"io"
	"testing"
)

func TestErrorf_SetsCodeAndMessage(t *testing.T) {
	e := Errorf(EBADREQUEST, "name %s missing", "email")
	if e.Code != EBADREQUEST {
		t.Errorf("Code = %q, want %q", e.Code, EBADREQUEST)
	}
	if e.Message != "name email missing" {
		t.Errorf("Message = %q, want %q", e.Message, "name email missing")
	}
	if e.Cause != nil {
		t.Errorf("Cause = %v, want nil", e.Cause)
	}
}

func TestErrorf_DefaultReportableFalse(t *testing.T) {
	if Errorf(EBADREQUEST, "x").Reportable {
		t.Errorf("Errorf Reportable = true, want false by default")
	}
	if Errorf(EINTERNAL, "x").Reportable {
		t.Errorf("Errorf Reportable = true, want false by default (caller sets it)")
	}
}

func TestWrap_SetsCause(t *testing.T) {
	e := Wrap(EINTERNAL, io.EOF, "read %s", "disk")
	if e.Code != EINTERNAL {
		t.Errorf("Code = %q, want %q", e.Code, EINTERNAL)
	}
	if e.Message != "read disk" {
		t.Errorf("Message = %q, want %q", e.Message, "read disk")
	}
	if e.Cause != io.EOF {
		t.Errorf("Cause = %v, want io.EOF", e.Cause)
	}
}

func TestWrap_UnwrapChain(t *testing.T) {
	e := Wrap(EINTERNAL, io.EOF, "oops")
	if !errors.Is(e, io.EOF) {
		t.Fatalf("errors.Is(Wrap(...), io.EOF) = false, want true")
	}
}

func TestErrorCode_Nil(t *testing.T) {
	if got := ErrorCode(nil); got != Code("") {
		t.Fatalf("ErrorCode(nil) = %q, want empty", got)
	}
}

func TestErrorCode_ErrsError(t *testing.T) {
	e := Errorf(EFORBIDDEN, "")
	if got := ErrorCode(e); got != EFORBIDDEN {
		t.Fatalf("ErrorCode = %q, want %q", got, EFORBIDDEN)
	}
}

func TestErrorCode_WrappedErrsError(t *testing.T) {
	e := fmt.Errorf("ctx: %w", Errorf(EFORBIDDEN, ""))
	if got := ErrorCode(e); got != EFORBIDDEN {
		t.Fatalf("ErrorCode(wrapped) = %q, want %q", got, EFORBIDDEN)
	}
}

func TestErrorCode_UnknownError(t *testing.T) {
	if got := ErrorCode(io.EOF); got != EINTERNAL {
		t.Fatalf("ErrorCode(io.EOF) = %q, want %q", got, EINTERNAL)
	}
}

func TestErrorMessage_Nil(t *testing.T) {
	if got := ErrorMessage(nil); got != "" {
		t.Fatalf("ErrorMessage(nil) = %q, want empty", got)
	}
}

func TestErrorMessage_UnknownError(t *testing.T) {
	if got := ErrorMessage(io.EOF); got != "Internal error." {
		t.Fatalf("ErrorMessage(io.EOF) = %q, want %q", got, "Internal error.")
	}
}

func TestErrorMessage_ErrsError(t *testing.T) {
	e := Errorf(EBADREQUEST, "bad %s", "input")
	if got := ErrorMessage(e); got != "bad input" {
		t.Fatalf("ErrorMessage = %q, want %q", got, "bad input")
	}
}
