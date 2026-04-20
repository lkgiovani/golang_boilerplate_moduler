package healthdomain

import "golang_boilerplate_module/internal/shared/domain/errs"

// DatabaseUnavailable is returned by the readiness check when the DB
// ping fails. The underlying driver error is wrapped in Cause for
// observability. Reportable=true so the error_handler logs it.
func DatabaseUnavailable(cause error) *errs.Error {
	e := errs.Wrap(errs.EUNAVAILABLE, cause, "Database not ready")
	e.Reportable = true
	return e
}
