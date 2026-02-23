package exceptions

type ExceptionCode string

const (
	CodeBadRequest         ExceptionCode = "BAD_REQUEST"
	CodeUnauthorized       ExceptionCode = "UNAUTHORIZED"
	CodeForbidden          ExceptionCode = "FORBIDDEN"
	CodeNotFound           ExceptionCode = "NOT_FOUND"
	CodeUnprocessable      ExceptionCode = "UNPROCESSABLE"
	CodeInternal           ExceptionCode = "INTERNAL"
	CodeServiceUnavailable ExceptionCode = "SERVICE_UNAVAILABLE"
)
