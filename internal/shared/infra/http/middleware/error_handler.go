package middleware

import (
	"errors"

	"golang_boilerplate_module/internal/shared/domain/errs"
	"golang_boilerplate_module/internal/shared/domain/providers"

	"github.com/gofiber/fiber/v2"
)

// httpErrorLabel maps an errs.Code to the standard HTTP error label
// returned in the "error" field of the response body. Kept private to
// this file since labels are HTTP-specific and not part of the domain.
var httpErrorLabel = map[errs.Code]string{
	errs.EBADREQUEST:     "Bad Request",
	errs.EINVALID:        "Unprocessable Entity",
	errs.EUNAUTHORIZED:   "Unauthorized",
	errs.EFORBIDDEN:      "Forbidden",
	errs.ENOTFOUND:       "Not Found",
	errs.ECONFLICT:       "Conflict",
	errs.EDUPLICATION:    "Conflict",
	errs.EPRECONDITION:   "Precondition Failed",
	errs.EEXPIRED:        "Gone",
	errs.ERATELIMIT:      "Too Many Requests",
	errs.ENOTIMPLEMENTED: "Not Implemented",
	errs.ETIMEOUT:        "Gateway Timeout",
	errs.EUNAVAILABLE:    "Service Unavailable",
	errs.EINTERNAL:       "Internal Server Error",
}

type errorResponse struct {
	Status  int    `json:"status"`
	Error   string `json:"error"`
	Message string `json:"message"`
}

// NewErrorHandler returns the Fiber error handler used by the API.
// It converts any error into an *errs.Error (wrapping fiber/unknown errors
// into EINTERNAL), logs Reportable errors at error level, and writes a
// JSON response with {status, error, message}. The Cause chain and
// Metadata are NEVER exposed in the HTTP response body (T-8-01, T-8-02).
func NewErrorHandler(rootLogger providers.LoggerProvider) fiber.ErrorHandler {
	return func(c *fiber.Ctx, err error) error {
		domainErr := toDomainError(err)

		if domainErr.Reportable {
			logger := LoggerFromLocals(c, rootLogger)
			logger.Error(domainErr.Message,
				"code", string(domainErr.Code),
				"metadata", domainErr.Metadata,
			)
		}

		status := errs.HTTPStatus(domainErr.Code)
		label, ok := httpErrorLabel[domainErr.Code]
		if !ok {
			label = httpErrorLabel[errs.EINTERNAL]
		}

		return c.Status(status).JSON(errorResponse{
			Status:  status,
			Error:   label,
			Message: domainErr.Message,
		})
	}
}

// toDomainError normalizes any error into an *errs.Error.
// Nil-cause fiber errors map to ENOTFOUND (404) or EINTERNAL otherwise.
// Unknown errors are wrapped in EINTERNAL with Reportable=true so they
// log and record at error level.
func toDomainError(err error) *errs.Error {
	var e *errs.Error
	if errors.As(err, &e) {
		return e
	}

	var fiberErr *fiber.Error
	if errors.As(err, &fiberErr) {
		if fiberErr.Code == fiber.StatusNotFound {
			return errs.Errorf(errs.ENOTFOUND, "Not found")
		}
		wrapped := errs.Wrap(errs.EINTERNAL, fiberErr, "Internal server error")
		wrapped.Reportable = true
		wrapped.Metadata = map[string]any{
			"fiberCode": fiberErr.Code,
		}
		return wrapped
	}

	wrapped := errs.Wrap(errs.EINTERNAL, err, "Internal server error")
	wrapped.Reportable = true
	return wrapped
}
