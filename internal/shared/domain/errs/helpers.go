package errs

import (
	"errors"
	"fmt"
)

// Errorf constructs a new *Error with a formatted message and no cause.
// Primary constructor for domain errors declared in per-module factories.
func Errorf(code Code, format string, args ...any) *Error {
	return &Error{
		Code:    code,
		Message: fmt.Sprintf(format, args...),
	}
}

// Wrap constructs a new *Error that wraps cause. errors.Is and errors.As
// walk through the Cause via (*Error).Unwrap.
func Wrap(code Code, cause error, format string, args ...any) *Error {
	return &Error{
		Code:    code,
		Message: fmt.Sprintf(format, args...),
		Cause:   cause,
	}
}

// ErrorCode extracts the Code from any error in the chain.
// Returns "" on nil, EINTERNAL on unknown error types.
func ErrorCode(err error) Code {
	if err == nil {
		return ""
	}
	var e *Error
	if errors.As(err, &e) {
		return e.Code
	}
	return EINTERNAL
}

// ErrorMessage extracts the Message from any error in the chain.
// Returns "" on nil, "Internal error." on unknown error types.
func ErrorMessage(err error) string {
	if err == nil {
		return ""
	}
	var e *Error
	if errors.As(err, &e) {
		return e.Message
	}
	return "Internal error."
}
