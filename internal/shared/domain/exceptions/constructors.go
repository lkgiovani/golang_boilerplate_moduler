package exceptions

func NewBadRequestException(message string, metadata map[string]any) *DomainError {
	if message == "" {
		message = "Bad request"
	}
	return &DomainError{
		Code:     CodeBadRequest,
		Message:  message,
		Metadata: metadata,
	}
}

func NewUnauthorizedException(message string, metadata map[string]any) *DomainError {
	if message == "" {
		message = "Unauthorized"
	}
	return &DomainError{
		Code:     CodeUnauthorized,
		Message:  message,
		Metadata: metadata,
	}
}

func NewForbiddenException(message string, metadata map[string]any) *DomainError {
	if message == "" {
		message = "Forbidden"
	}
	return &DomainError{
		Code:     CodeForbidden,
		Message:  message,
		Metadata: metadata,
	}
}

func NewNotFoundException(message string, metadata map[string]any) *DomainError {
	if message == "" {
		message = "Not found"
	}
	return &DomainError{
		Code:     CodeNotFound,
		Message:  message,
		Metadata: metadata,
	}
}

func NewUnprocessableException(message string, metadata map[string]any) *DomainError {
	if message == "" {
		message = "Unprocessable entity"
	}
	return &DomainError{
		Code:     CodeUnprocessable,
		Message:  message,
		Metadata: metadata,
	}
}

func NewInternalException(metadata map[string]any) *DomainError {
	return &DomainError{
		Code:       CodeInternal,
		Message:    "Internal server error",
		Metadata:   metadata,
		Reportable: true,
	}
}

func NewServiceUnavailableException(message string, metadata map[string]any) *DomainError {
	if message == "" {
		message = "Service unavailable"
	}
	return &DomainError{
		Code:       CodeServiceUnavailable,
		Message:    message,
		Metadata:   metadata,
		Reportable: true,
	}
}
