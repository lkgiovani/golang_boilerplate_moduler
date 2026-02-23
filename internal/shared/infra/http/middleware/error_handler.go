package middleware

import (
	"errors"

	"golang_boilerplate_module/internal/shared/domain/exceptions"
	"golang_boilerplate_module/internal/shared/domain/providers"

	"github.com/gofiber/fiber/v2"
)

type httpMapping struct {
	Status int
	Error  string
}

var exceptionHTTPMap = map[exceptions.ExceptionCode]httpMapping{
	exceptions.CodeBadRequest:         {400, "Bad Request"},
	exceptions.CodeUnauthorized:       {401, "Unauthorized"},
	exceptions.CodeForbidden:          {403, "Forbidden"},
	exceptions.CodeNotFound:           {404, "Not Found"},
	exceptions.CodeUnprocessable:      {422, "Unprocessable Entity"},
	exceptions.CodeInternal:           {500, "Internal Server Error"},
	exceptions.CodeServiceUnavailable: {503, "Service Unavailable"},
}

type errorResponse struct {
	Status  int    `json:"status"`
	Error   string `json:"error"`
	Message string `json:"message"`
}

// NewErrorHandler returns a Fiber-compatible ErrorHandler closure.
// Equivalent to errorHandler.ts in the TypeScript project.
func NewErrorHandler(rootLogger providers.LoggerProvider) fiber.ErrorHandler {
	return func(c *fiber.Ctx, err error) error {
		var domainErr *exceptions.DomainError

		if !errors.As(err, &domainErr) {
			// Try Fiber's own error type (e.g. 404 from routing)
			var fiberErr *fiber.Error
			if errors.As(err, &fiberErr) {
				if fiberErr.Code == fiber.StatusNotFound {
					domainErr = exceptions.NewNotFoundException("", nil)
				} else {
					domainErr = exceptions.NewInternalException(map[string]any{
						"fiberCode": fiberErr.Code,
						"error":     fiberErr.Message,
					})
				}
			} else {
				domainErr = exceptions.NewInternalException(map[string]any{
					"error": err.Error(),
				})
			}
		}

		if domainErr.Reportable {
			logger := LoggerFromLocals(c, rootLogger)
			logger.Error(domainErr.Message,
				"code", domainErr.Code,
				"metadata", domainErr.Metadata,
			)
		}

		mapping, ok := exceptionHTTPMap[domainErr.Code]
		if !ok {
			mapping = exceptionHTTPMap[exceptions.CodeInternal]
		}

		return c.Status(mapping.Status).JSON(errorResponse{
			Status:  mapping.Status,
			Error:   mapping.Error,
			Message: domainErr.Message,
		})
	}
}
