package errs

import (
	"errors"
	"fmt"
)

// Error is the single domain error type used across all modules.
// It carries a typed Code, a human-readable Message, an optional Cause
// for wrapping, optional structured Metadata for logs/spans, and a
// Reportable flag read by the HTTP error handler to decide logging level.
type Error struct {
	Code       Code
	Message    string
	Cause      error
	Metadata   map[string]any
	Reportable bool
}

// Error implements the built-in error interface.
// Format matches the legacy DomainError format so log output stays stable.
func (e *Error) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap enables errors.Is / errors.As chain walking through Cause.
func (e *Error) Unwrap() error {
	return e.Cause
}

// Is enables `errors.Is(err, factory())` where factory() returns a fresh
// *Error instance. Matching is by Code, not pointer identity — this is
// the idiomatic Go pattern also used by os.ErrNotExist via syscall.Errno.
func (e *Error) Is(target error) bool {
	var t *Error
	if !errors.As(target, &t) {
		return false
	}
	return e.Code == t.Code
}
