package exceptions

import "fmt"

type DomainError struct {
	Code       ExceptionCode
	Message    string
	Metadata   map[string]any
	Reportable bool
}

func (e *DomainError) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}
